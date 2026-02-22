package consolehandler

import (
	"sync"
	"time"

	"github.com/philipp01105/nlog/core"
	"github.com/philipp01105/nlog/formatter"
	"github.com/philipp01105/nlog/handler"
)

// AsyncConsoleHandler is an asynchronous console handler with isolated
// queue and overflow logic. It avoids sync-path overhead and is optimized
// for parallel throughput with a dedicated background goroutine.
type AsyncConsoleHandler struct {
	consoleBase
	queue          chan *core.Entry
	wg             sync.WaitGroup
	overflowPolicy map[core.Level]handler.OverflowPolicy
	blockTimeout   time.Duration
	drainTimeout   time.Duration
	blockTimer     *time.Timer
	parBufPool     sync.Pool // pool of *parallelBuf for overflow fallback writes
}

// newAsyncConsoleHandler creates a new asynchronous console handler.
func newAsyncConsoleHandler(cfg ConsoleConfig) *AsyncConsoleHandler {
	h := &AsyncConsoleHandler{
		overflowPolicy: cfg.OverflowPolicy,
		blockTimeout:   cfg.BlockTimeout,
		drainTimeout:   cfg.DrainTimeout,
		blockTimer:     handler.NewStoppedTimer(),
	}
	h.writer = cfg.Writer
	h.formatter = cfg.Formatter
	h.concurrentSafe = cfg.ConcurrentWriter || isConcurrentSafeWriter(cfg.Writer)
	h.stats = handler.NewStats()
	h.closed = make(chan struct{})

	// Cache WriterFormatter for zero-alloc path
	h.writerFormatter, _ = cfg.Formatter.(formatter.WriterFormatter)

	// Cache BufferFormatter for sync fast path (avoids buffer pool + lockedWriter)
	h.bufferFormatter, _ = cfg.Formatter.(formatter.BufferFormatter)

	// Pre-allocate lockedWriter for lock-minimal write path
	h.lw = lockedWriter{mu: &h.mu, w: h.writer}

	// Pre-grow sync buffer for processWrite path
	if h.bufferFormatter != nil {
		h.syncBuf.Grow(256)
		h.parBufPool = sync.Pool{
			New: func() interface{} {
				pb := &parallelBuf{}
				pb.buf.Grow(256)
				pb.entry.Fields = make([]core.Field, 0, 16)
				return pb
			},
		}
	}

	h.queue = make(chan *core.Entry, cfg.BufferSize)
	h.wg.Add(1)
	go h.process()

	return h
}

// HandleLog processes log data by creating a pooled Entry and sending it
// to the async queue.
func (h *AsyncConsoleHandler) HandleLog(t time.Time, level core.Level, msg string, loggerFields, callFields []core.Field, caller core.CallerInfo) error {
	entry := core.GetEntry()
	entry.Time = t
	entry.Level = level
	entry.Message = msg
	entry.Caller = caller
	if len(loggerFields) > 0 {
		entry.Fields = append(entry.Fields, loggerFields...)
	}
	if len(callFields) > 0 {
		entry.Fields = append(entry.Fields, callFields...)
	}
	return h.Handle(entry)
}

// Handle sends a log entry to the async queue with overflow policy handling.
func (h *AsyncConsoleHandler) Handle(entry *core.Entry) error {
	// Get overflow policy for this level
	policy, ok := h.overflowPolicy[entry.Level]
	if !ok {
		policy = handler.DropNewest // Default if not specified
	}

	switch policy {
	case handler.Block:
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
				return h.write(entry, &h.parBufPool)
			case <-h.closed:
				// Handler is closing, write synchronously
				if !h.blockTimer.Stop() {
					select {
					case <-h.blockTimer.C:
					default:
					}
				}
				return h.write(entry, &h.parBufPool)
			}
		}

	case handler.DropOldest:
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

	case handler.DropNewest:
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

// CanRecycleEntry returns false because the async handler processes entries
// in a background goroutine after Handle returns.
func (h *AsyncConsoleHandler) CanRecycleEntry() bool {
	return false
}

// process handles async log processing
func (h *AsyncConsoleHandler) process() {
	defer h.wg.Done()

	for {
		select {
		case entry := <-h.queue:
			err := h.processWrite(entry, &h.parBufPool)
			if err != nil {
				return
			}
			core.PutEntry(entry)
			// Batch drain: process additional queued entries without blocking
		batchDrain:
			for {
				select {
				case entry := <-h.queue:
					err := h.processWrite(entry, &h.parBufPool)
					if err != nil {
						return
					}
					core.PutEntry(entry)
				default:
					break batchDrain
				}
			}
		case <-h.closed:
			// Drain remaining entries with timeout
			deadline := time.After(h.drainTimeout)
		drainLoop:
			for {
				select {
				case entry := <-h.queue:
					err := h.processWrite(entry, &h.parBufPool)
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

// Close closes the handler, draining the queue with a timeout.
func (h *AsyncConsoleHandler) Close() error {
	// Check if already closed (without lock to avoid deadlock)
	select {
	case <-h.closed:
		return nil // Already closed
	default:
	}

	close(h.closed)
	h.wg.Wait() // Wait without holding lock to avoid deadlock

	h.mu.Lock()
	close(h.queue)
	h.mu.Unlock()

	return nil
}
