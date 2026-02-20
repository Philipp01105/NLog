package handler

import (
	"time"

	"github.com/philipp01105/nlog/core"
)

// MultiHandler sends log entries to multiple handlers
type MultiHandler struct {
	handlers     []Handler
	fastHandlers []FastHandler // cached FastHandler interfaces (nil when handler doesn't implement it)
	allFast      bool          // true when every child implements FastHandler
	recycleEntry bool          // true when every child supports entry recycling
}

// NewMultiHandler creates a new multi-handler
func NewMultiHandler(handlers ...Handler) *MultiHandler {
	m := &MultiHandler{
		handlers:     handlers,
		fastHandlers: make([]FastHandler, len(handlers)),
		allFast:      true,
		recycleEntry: true,
	}
	for i, h := range handlers {
		if fh, ok := h.(FastHandler); ok {
			m.fastHandlers[i] = fh
		} else {
			m.allFast = false
		}
		if rc, ok := h.(interface{ CanRecycleEntry() bool }); ok {
			if !rc.CanRecycleEntry() {
				m.recycleEntry = false
			}
		} else {
			m.recycleEntry = false
		}
	}
	return m
}

// HandleLog processes log data directly without requiring a pooled Entry.
// When all children implement FastHandler, this avoids Entry allocation entirely.
func (h *MultiHandler) HandleLog(t time.Time, level core.Level, msg string, loggerFields, callFields []core.Field, caller core.CallerInfo) error {
	if h.allFast {
		var lastErr error
		for _, fh := range h.fastHandlers {
			if err := fh.HandleLog(t, level, msg, loggerFields, callFields, caller); err != nil {
				lastErr = err
			}
		}
		return lastErr
	}

	// Mixed path: build a pooled entry for non-fast handlers
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
	var lastErr error
	for i, handler := range h.handlers {
		if fh := h.fastHandlers[i]; fh != nil {
			if err := fh.HandleLog(t, level, msg, loggerFields, callFields, caller); err != nil {
				lastErr = err
			}
		} else if err := handler.Handle(entry); err != nil {
			lastErr = err
		}
	}
	if h.recycleEntry {
		core.PutEntry(entry)
	}
	return lastErr
}

// Handle processes a log entry by sending it to all handlers
func (h *MultiHandler) Handle(entry *core.Entry) error {
	var lastErr error
	for _, handler := range h.handlers {
		if err := handler.Handle(entry); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// CanRecycleEntry returns true if the caller can recycle the entry after Handle returns.
// This is safe when all child handlers process entries synchronously.
func (h *MultiHandler) CanRecycleEntry() bool {
	return h.recycleEntry
}

// Close closes all handlers
func (h *MultiHandler) Close() error {
	var lastErr error
	for _, handler := range h.handlers {
		if err := handler.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
