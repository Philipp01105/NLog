package handler

import "github.com/Philipp01105/NLog/core"

// Handler defines the interface for log handlers
type Handler interface {
	// Handle processes a log entry
	Handle(entry *core.Entry) error

	// Close closes the handler and releases resources
	Close() error
}
