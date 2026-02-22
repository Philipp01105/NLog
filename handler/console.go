package handler

import (
	"bytes"
	"io"
	"os"
	"sync"
	"time"

	"github.com/philipp01105/nlog/core"
	"github.com/philipp01105/nlog/formatter"
)

// lockedWriter wraps an io.Writer with a mutex, acquiring the lock only
// for Write calls. Formatters prepare data in their own pooled buffers
// and call Write once, so the lock is held only during the actual I/O.
type lockedWriter struct {
	mu *sync.Mutex
	w  io.Writer
}

func (lw *lockedWriter) Write(p []byte) (n int, err error) {
	lw.mu.Lock()
	n, err = lw.w.Write(p)
	lw.mu.Unlock()
	return
}

// parallelBuf combines an entry and buffer for pool-friendly parallel formatting.
// Pooling them together reduces HandleLog's parallel fallback from 4 pool
// operations (entry pool Get/Put + formatter buffer Get/Put) to 2.
type parallelBuf struct {
	buf   bytes.Buffer
	entry core.Entry
}

// ConsoleHandler writes log entries to stdout/stderr
type ConsoleHandler struct {
	writer          io.Writer
	formatter       formatter.Formatter
	writerFormatter formatter.WriterFormatter
	bufferFormatter formatter.BufferFormatter
	async           bool
	queue           chan *core.Entry
	wg              sync.WaitGroup
	closed          chan struct{}
	mu              sync.Mutex // protects syncBuf, syncEntry (format lock)
	writeMu         sync.Mutex // protects writer (I/O lock, held briefly)
	// Lock ordering: always mu before writeMu. Never acquire mu while holding writeMu.
	lw             lockedWriter
	syncBuf        bytes.Buffer
	syncEntry      core.Entry
	parBufPool     sync.Pool // pool of *parallelBuf for parallel HandleLog path
	overflowPolicy map[core.Level]OverflowPolicy
	blockTimeout   time.Duration
	stats          *Stats
	drainTimeout   time.Duration
	blockTimer     *time.Timer
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
		blockTimer:     NewStoppedTimer(),
	}

	// Cache WriterFormatter for zero-alloc path
	h.writerFormatter, _ = cfg.Formatter.(formatter.WriterFormatter)

	// Cache BufferFormatter for sync fast path (avoids buffer pool + lockedWriter)
	h.bufferFormatter, _ = cfg.Formatter.(formatter.BufferFormatter)

	// Pre-allocate lockedWriter for lock-minimal write path
	h.lw = lockedWriter{mu: &h.writeMu, w: h.writer}

	// Pre-grow sync buffer for handler-owned format path
	if h.bufferFormatter != nil {
		h.syncBuf.Grow(256)
		h.syncEntry.Fields = make([]core.Field, 0, 16)
		h.parBufPool = sync.Pool{
			New: func() interface{} {
				pb := &parallelBuf{}
				pb.buf.Grow(256)
				pb.entry.Fields = make([]core.Field, 0, 16)
				return pb
			},
		}
	}

	if h.async {
		h.queue = make(chan *core.Entry, cfg.BufferSize)
		h.wg.Add(1)
		go h.process()
	}

	return h
}

// HandleLog processes log data directly without requiring a pooled Entry.
// Under no contention, uses handler-owned buffer for zero-alloc formatting.
// Under contention (parallel callers), uses a combined entry+buffer pool
// that formats outside the format lock for better parallel throughput.
func (h *ConsoleHandler) HandleLog(t time.Time, level core.Level, msg string, loggerFields, callFields []core.Field, caller core.CallerInfo) error {
	if !h.async && h.bufferFormatter != nil {
		if h.mu.TryLock() {
			h.syncEntry.Time = t
			h.syncEntry.Level = level
			h.syncEntry.Message = msg
			// Caller is always set by the logger: either GetCaller() result or zero value
			h.syncEntry.Caller = caller
			h.syncEntry.Fields = h.syncEntry.Fields[:0]
			if len(loggerFields) > 0 {
				h.syncEntry.Fields = append(h.syncEntry.Fields, loggerFields...)
			}
			if len(callFields) > 0 {
				h.syncEntry.Fields = append(h.syncEntry.Fields, callFields...)
			}

			h.syncBuf.Reset()
			h.bufferFormatter.FormatEntry(&h.syncEntry, &h.syncBuf)
			h.writeMu.Lock()
			_, err := h.writer.Write(h.syncBuf.Bytes())
			h.writeMu.Unlock()
			h.mu.Unlock()
			if err == nil {
				h.stats.IncrementProcessed()
			}
			return err
		}

		// Parallel fallback: combined entry+buffer from pool avoids
		// Entry pool Get/Put + formatter buffer Get/Put (2 ops vs 4)
		// and skips the second TryLock attempt in write().
		pb := h.parBufPool.Get().(*parallelBuf)
		pb.entry.Time = t
		pb.entry.Level = level
		pb.entry.Message = msg
		pb.entry.Caller = caller
		pb.entry.Fields = pb.entry.Fields[:0]
		if len(loggerFields) > 0 {
			pb.entry.Fields = append(pb.entry.Fields, loggerFields...)
		}
		if len(callFields) > 0 {
			pb.entry.Fields = append(pb.entry.Fields, callFields...)
		}

		pb.buf.Reset()
		h.bufferFormatter.FormatEntry(&pb.entry, &pb.buf)
		h.writeMu.Lock()
		_, err := h.writer.Write(pb.buf.Bytes())
		h.writeMu.Unlock()

		// Clean for pool reuse
		pb.entry.Fields = pb.entry.Fields[:0]
		if pb.entry.Caller.Defined {
			pb.entry.Caller = core.CallerInfo{}
		}
		h.parBufPool.Put(pb)

		if err == nil {
			h.stats.IncrementProcessed()
		}
		return err
	}

	// Fallback for async mode or non-BufferFormatter: pool entry + Handle
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
	err := h.Handle(entry)
	if !h.async {
		core.PutEntry(entry)
	}
	return err
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

// write formats and writes an entry.
// Uses TryLock on mu to access handler-owned buffer when uncontended (zero pool
// overhead). When contended, falls through to writerFormatter which formats
// in a pool buffer and locks writeMu only for the final I/O – minimizing
// lock hold time for parallel callers.
func (h *ConsoleHandler) write(entry *core.Entry) error {
	if h.bufferFormatter != nil {
		if h.mu.TryLock() {
			h.syncBuf.Reset()
			h.bufferFormatter.FormatEntry(entry, &h.syncBuf)
			h.writeMu.Lock()
			_, err := h.writer.Write(h.syncBuf.Bytes())
			h.writeMu.Unlock()
			h.mu.Unlock()
			if err == nil {
				h.stats.IncrementProcessed()
			}
			return err
		}
	}

	if h.writerFormatter != nil {
		// FormatTo prepares data in an internal pooled buffer, then calls
		// lw.Write once – writeMu is held only for the final I/O write.
		err := h.writerFormatter.FormatTo(entry, &h.lw)
		if err == nil {
			h.stats.IncrementProcessed()
		}
		return err
	}

	data, err := h.formatter.Format(entry)
	if err != nil {
		return err
	}

	h.writeMu.Lock()
	_, writeErr := h.writer.Write(data)
	h.writeMu.Unlock()

	if writeErr == nil {
		h.stats.IncrementProcessed()
	}

	return writeErr
}

// processWrite formats and writes using handler-owned buffer under Lock.
// Used only by the single-consumer process() goroutine where contention
// is impossible, so the lock always succeeds immediately.
func (h *ConsoleHandler) processWrite(entry *core.Entry) error {
	if h.bufferFormatter != nil {
		h.mu.Lock()
		h.syncBuf.Reset()
		h.bufferFormatter.FormatEntry(entry, &h.syncBuf)
		h.writeMu.Lock()
		_, err := h.writer.Write(h.syncBuf.Bytes())
		h.writeMu.Unlock()
		h.mu.Unlock()
		if err == nil {
			h.stats.IncrementProcessed()
		}
		return err
	}
	return h.write(entry)
}

// CanRecycleEntry returns true if the caller can recycle the entry after Handle returns
func (h *ConsoleHandler) CanRecycleEntry() bool {
	return !h.async
}

// process handles async log processing
func (h *ConsoleHandler) process() {
	defer h.wg.Done()

	// batchBuf accumulates formatted entries so we can issue a single Write
	// call per batch instead of one syscall per entry.
	var batchBuf bytes.Buffer
	batchBuf.Grow(4096)

	for {
		select {
		case entry := <-h.queue:
			if h.bufferFormatter != nil {
				// Batch path: format all currently-queued entries into batchBuf,
				// then issue a single Write call for the entire batch.
				batchBuf.Reset()
				h.bufferFormatter.FormatEntry(entry, &batchBuf)
				core.PutEntry(entry)
				batchCount := 1
			batchDrain:
				for {
					select {
					case entry := <-h.queue:
						h.bufferFormatter.FormatEntry(entry, &batchBuf)
						core.PutEntry(entry)
						batchCount++
					default:
						break batchDrain
					}
				}
				// Count entries as processed before writing: they have already been
				// dequeued and recycled via PutEntry, so they are consumed
				// regardless of whether the Write call succeeds.
				h.stats.AddProcessed(uint64(batchCount))
				h.writeMu.Lock()
				_, writeErr := h.writer.Write(batchBuf.Bytes())
				h.writeMu.Unlock()
				if writeErr != nil {
					return
				}
			} else {
				// Non-bufferFormatter fallback: individual write per entry.
				if err := h.processWrite(entry); err != nil {
					return
				}
				core.PutEntry(entry)
			drainFallback:
				for {
					select {
					case entry := <-h.queue:
						if err := h.processWrite(entry); err != nil {
							return
						}
						core.PutEntry(entry)
					default:
						break drainFallback
					}
				}
			}
		case <-h.closed:
			// Drain remaining entries with timeout
			deadline := time.After(h.drainTimeout)
		drainLoop:
			for {
				select {
				case entry := <-h.queue:
					if err := h.processWrite(entry); err != nil {
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
