package filehandler

import (
	"os"
	"time"

	"github.com/philipp01105/nlog/core"
)

// SyncFileHandler is a synchronous file handler optimized for the hot path.
// It avoids async queue overhead and eliminates branches that would be needed
// to support both sync and async modes.
type SyncFileHandler struct {
	fileBase
	syncEntry core.Entry
}

// newSyncFileHandler creates a new synchronous file handler.
func newSyncFileHandler(cfg FileConfig, file *os.File, fileSize int64) *SyncFileHandler {
	h := &SyncFileHandler{}
	initFileBase(&h.fileBase, cfg, file, fileSize)
	// Pre-allocate syncEntry fields if bufferFormatter is available
	if h.bufferFormatter != nil {
		h.syncEntry.Fields = make([]core.Field, 0, 16)
	}
	return h
}

// HandleLog processes log data directly without requiring a pooled Entry.
// This avoids sync.Pool Get/Put overhead for the sync fast path.
func (h *SyncFileHandler) HandleLog(t time.Time, level core.Level, msg string, loggerFields, callFields []core.Field, caller core.CallerInfo) error {
	if h.bufferFormatter != nil {
		h.mu.Lock()
		if err := h.rotateIfNeeded(); err != nil {
			h.mu.Unlock()
			return err
		}
		h.syncEntry.Time = t
		h.syncEntry.Level = level
		h.syncEntry.Message = msg
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
		n, err := h.bufWriter.Write(h.syncBuf.Bytes())
		if err == nil {
			h.currentSize += int64(n)
			h.stats.IncrementProcessed()
		}
		h.mu.Unlock()
		return err
	}

	// Fallback: create a pooled entry and use Handle
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
func (h *SyncFileHandler) Handle(entry *core.Entry) error {
	return h.write(entry)
}

// CanRecycleEntry returns true because sync handler processes entries immediately.
func (h *SyncFileHandler) CanRecycleEntry() bool {
	return true
}

// Close closes the handler and the underlying file.
func (h *SyncFileHandler) Close() error {
	select {
	case <-h.closed:
		return nil // Already closed
	default:
		close(h.closed)
	}
	return h.closeFile()
}
