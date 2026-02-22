package consolehandler

import (
	"bytes"
	"sync"
	"time"

	"github.com/philipp01105/nlog/core"
	"github.com/philipp01105/nlog/formatter"
	"github.com/philipp01105/nlog/handler"
)

// SyncConsoleHandler is a synchronous console handler optimized for the
// single-goroutine hot path. It avoids async queue overhead and eliminates
// branches that would be needed to support both sync and async modes.
type SyncConsoleHandler struct {
	consoleBase
	syncEntry  core.Entry
	parBufPool sync.Pool // pool of *parallelBuf for parallel HandleLog path
}

// newSyncConsoleHandler creates a new synchronous console handler.
func newSyncConsoleHandler(cfg ConsoleConfig) *SyncConsoleHandler {
	h := &SyncConsoleHandler{}
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

	return h
}

// HandleLog processes log data directly without requiring a pooled Entry.
// Under no contention, uses handler-owned buffer for zero-alloc formatting.
// Under contention (parallel callers), uses a combined entry+buffer pool
// that formats outside the format lock for better parallel throughput.
func (h *SyncConsoleHandler) HandleLog(t time.Time, level core.Level, msg string, loggerFields, callFields []core.Field, caller core.CallerInfo) error {
	if h.bufferFormatter != nil {
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
			// Write under mu: already held, serializes all writes.
			_, err := h.writer.Write(h.syncBuf.Bytes())
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
		var err error
		if h.concurrentSafe {
			_, err = h.writer.Write(pb.buf.Bytes())
		} else {
			h.mu.Lock()
			_, err = h.writer.Write(pb.buf.Bytes())
			h.mu.Unlock()
		}

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

	// Fallback for non-BufferFormatter: pool entry + Handle
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
	core.PutEntry(entry)
	return err
}

// Handle processes a log entry synchronously.
func (h *SyncConsoleHandler) Handle(entry *core.Entry) error {
	return h.write(entry, &h.parBufPool)
}

// CanRecycleEntry returns true because sync handler processes entries immediately.
func (h *SyncConsoleHandler) CanRecycleEntry() bool {
	return true
}

// Close closes the handler.
func (h *SyncConsoleHandler) Close() error {
	select {
	case <-h.closed:
		return nil // Already closed
	default:
		close(h.closed)
	}
	return nil
}

// parallelBuf combines an entry and buffer for pool-friendly parallel formatting.
// Pooling them together reduces HandleLog's parallel fallback from 4 pool
// operations (entry pool Get/Put + formatter buffer Get/Put) to 2.
type parallelBuf struct {
	buf   bytes.Buffer
	entry core.Entry
}
