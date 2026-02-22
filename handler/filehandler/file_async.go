package filehandler

import (
	"os"
	"sync"
	"time"

	"github.com/philipp01105/nlog/core"
	"github.com/philipp01105/nlog/handler"
)

// AsyncFileHandler is an asynchronous file handler with isolated queue
// and overflow logic. It avoids sync-path overhead and is optimized for
// parallel throughput with a dedicated background goroutine.
type AsyncFileHandler struct {
	fileBase
	queue          chan *core.Entry
	wg             sync.WaitGroup
	overflowPolicy map[core.Level]handler.OverflowPolicy
	blockTimeout   time.Duration
	drainTimeout   time.Duration
	blockTimer     *time.Timer
}

// newAsyncFileHandler creates a new asynchronous file handler.
func newAsyncFileHandler(cfg FileConfig, file *os.File, fileSize int64) *AsyncFileHandler {
	h := &AsyncFileHandler{
		overflowPolicy: cfg.OverflowPolicy,
		blockTimeout:   cfg.BlockTimeout,
		drainTimeout:   cfg.DrainTimeout,
		blockTimer:     handler.NewStoppedTimer(),
	}
	initFileBase(&h.fileBase, cfg, file, fileSize)

	h.queue = make(chan *core.Entry, cfg.BufferSize)
	h.wg.Add(1)
	go h.process()

	return h
}

// HandleLog processes log data by creating a pooled Entry and sending it
// to the async queue.
func (h *AsyncFileHandler) HandleLog(t time.Time, level core.Level, msg string, loggerFields, callFields []core.Field, caller core.CallerInfo) error {
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
func (h *AsyncFileHandler) Handle(entry *core.Entry) error {
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
func (h *AsyncFileHandler) CanRecycleEntry() bool {
	return false
}

// process handles async log processing
func (h *AsyncFileHandler) process() {
	defer h.wg.Done()

	for {
		select {
		case entry := <-h.queue:
			err := h.write(entry)
			if err != nil {
				return
			}
			core.PutEntry(entry)
			// Batch drain: process additional queued entries without blocking
		batchDrain:
			for {
				select {
				case entry := <-h.queue:
					err := h.write(entry)
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

// Close closes the handler, draining the queue with a timeout.
func (h *AsyncFileHandler) Close() error {
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

	return h.closeFile()
}
