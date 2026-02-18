package handler

import (
	"github.com/Philipp01105/logging-framework/core"
)

// MultiHandler sends log entries to multiple handlers
type MultiHandler struct {
	handlers []Handler
}

// NewMultiHandler creates a new multi-handler
func NewMultiHandler(handlers ...Handler) *MultiHandler {
	return &MultiHandler{
		handlers: handlers,
	}
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
