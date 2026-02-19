package handler

import (
	"io"
	"os"
	"sync"
	"time"

	"github.com/philipp01105/nlog/core"
	"github.com/philipp01105/nlog/formatter"
)

// ConsoleHandler writes log entries to stdout/stderr
type ConsoleHandler struct {
	writer          io.Writer
	formatter       formatter.Formatter
	writerFormatter formatter.WriterFormatter
	async           bool
	queue           chan *core.Entry
	wg              sync.WaitGroup
	closed          chan struct{}
	mu              sync.Mutex
	overflowPolicy  map[core.Level]OverflowPolicy
	blockTimeout    time.Duration
	stats           *Stats
	drainTimeout    time.Duration
	blockTimer      *time.Timer
}

// ConsoleConfig holds configuration for console handler
type ConsoleConfig struct {
	// Writer to write to (default: os.Stdout)
	Writer io.Writer
	// Formatter to use (default: TextFormatter)
	Formatter formatter.Formatter
	// Async enables asynchronous logging (default: true)
	Async bool
	// BufferSize is the size of the async queue (default: 1000)
	BufferSize int
	// OverflowPolicy defines per-level overflow behavior (default: uses DefaultLevelPolicy)
	OverflowPolicy map[core.Level]OverflowPolicy
	// BlockTimeout is the timeout for blocking overflow policy (default: 100ms)
	BlockTimeout time.Duration
	// DrainTimeout is the timeout for draining queue on Close (default: 5s)
	DrainTimeout time.Duration
}

// NewConsoleHandler creates a new console handler
func NewConsoleHandler(cfg ConsoleConfig) *ConsoleHandler {
	if cfg.Writer == nil {
		cfg.Writer = os.Stdout
	}
	if cfg.Formatter == nil {
		cfg.Formatter = formatter.NewTextFormatter(formatter.Config{})
	}
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 1000
	}
	if cfg.OverflowPolicy == nil {
		cfg.OverflowPolicy = DefaultLevelPolicy()
	}
	if cfg.BlockTimeout == 0 {
		cfg.BlockTimeout = 100 * time.Millisecond
	}
	if cfg.DrainTimeout == 0 {
		cfg.DrainTimeout = 5 * time.Second
	}

	h := &ConsoleHandler{
		writer:         cfg.Writer,
		formatter:      cfg.Formatter,
		async:          cfg.Async,
		closed:         make(chan struct{}),
		overflowPolicy: cfg.OverflowPolicy,
		blockTimeout:   cfg.BlockTimeout,
		stats:          NewStats(),
		drainTimeout:   cfg.DrainTimeout,
		blockTimer:     newStoppedTimer(),
	}

	// Cache WriterFormatter for zero-alloc path
	h.writerFormatter, _ = cfg.Formatter.(formatter.WriterFormatter)

	if h.async {
		h.queue = make(chan *core.Entry, cfg.BufferSize)
		h.wg.Add(1)
		go h.process()
	}

	return h
}

// Handle processes a log entry
func (h *ConsoleHandler) Handle(entry *core.Entry) error {
	if !h.async {
		return h.write(entry)
	}

	// Get overflow policy for this level
	policy, ok := h.overflowPolicy[entry.Level]
	if !ok {
		policy = DropNewest // Default if not specified
	}

	switch policy {
	case Block:
		// Try to send with timeout using reusable timer
		select {
		case h.queue <- entry:
			return nil
		default:
			// Queue full, use timer for timeout
			if !h.blockTimer.Stop() {
				select {
				case <-h.blockTimer.C:
				default:
				}
			}
			h.blockTimer.Reset(h.blockTimeout)
			select {
			case h.queue <- entry:
				if !h.blockTimer.Stop() {
					select {
					case <-h.blockTimer.C:
					default:
					}
				}
				return nil
			case <-h.blockTimer.C:
				// Timeout - fall back to synchronous write
				h.stats.IncrementBlocked()
				return h.write(entry)
			case <-h.closed:
				// Handler is closing, write synchronously
				if !h.blockTimer.Stop() {
					select {
					case <-h.blockTimer.C:
					default:
					}
				}
				return h.write(entry)
			}
		}

	case DropOldest:
		// Try non-blocking send
		select {
		case h.queue <- entry:
			return nil
		default:
			// Queue full - try to drop oldest
			select {
			case <-h.queue: // Remove oldest
				h.stats.IncrementDropped(entry.Level)
			default:
			}
			// Try again
			select {
			case h.queue <- entry:
				return nil
			default:
				// Still full, drop this one
				h.stats.IncrementDropped(entry.Level)
				return nil
			}
		}

	case DropNewest:
		fallthrough
	default:
		// Non-blocking send
		select {
		case h.queue <- entry:
			return nil
		default:
			// Queue full - drop this entry
			h.stats.IncrementDropped(entry.Level)
			return nil
		}
	}
}

// write formats and writes an entry
func (h *ConsoleHandler) write(entry *core.Entry) error {
	if h.writerFormatter != nil {
		h.mu.Lock()
		err := h.writerFormatter.FormatTo(entry, h.writer)
		h.mu.Unlock()
		if err == nil {
			h.stats.IncrementProcessed()
		}
		return err
	}

	data, err := h.formatter.Format(entry)
	if err != nil {
		return err
	}

	h.mu.Lock()
	_, writeErr := h.writer.Write(data)
	h.mu.Unlock()

	if writeErr == nil {
		h.stats.IncrementProcessed()
	}

	return writeErr
}

// CanRecycleEntry returns true if the caller can recycle the entry after Handle returns
func (h *ConsoleHandler) CanRecycleEntry() bool {
	return !h.async
}

// process handles async log processing
func (h *ConsoleHandler) process() {
	defer h.wg.Done()

	for {
		select {
		case entry := <-h.queue:
			err := h.write(entry)
			if err != nil {
				return
			}
			core.PutEntry(entry)
		case <-h.closed:
			// Drain remaining entries with timeout
			deadline := time.After(h.drainTimeout)
		drainLoop:
			for {
				select {
				case entry := <-h.queue:
					err := h.write(entry)
					if err != nil {
						return
					}
					core.PutEntry(entry)
				case <-deadline:
					// Timeout reached, stop draining
					break drainLoop
				default:
					// Queue empty
					break drainLoop
				}
			}
			return
		}
	}
}

// Stats returns a snapshot of the current statistics
func (h *ConsoleHandler) Stats() Snapshot {
	return h.stats.GetSnapshot()
}

// Close closes the handler
func (h *ConsoleHandler) Close() error {
	// Check if already closed (without lock to avoid deadlock)
	select {
	case <-h.closed:
		return nil // Already closed
	default:
	}

	if h.async {
		close(h.closed)
		h.wg.Wait() // Wait without holding lock to avoid deadlock

		h.mu.Lock()
		close(h.queue)
		h.mu.Unlock()
	}
	return nil
}
