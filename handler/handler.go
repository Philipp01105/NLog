package handler

import (
	"time"

	"github.com/philipp01105/nlog/core"
)

// Handler defines the interface for log handlers
type Handler interface {
	// Handle processes a log entry
	Handle(entry *core.Entry) error

	// Close closes the handler and releases resources
	Close() error
}

// FastHandler is an optional interface that handlers can implement
// to process log data directly without requiring an Entry from the pool.
type FastHandler interface {
	HandleLog(t time.Time, level core.Level, msg string, loggerFields, callFields []core.Field, caller core.CallerInfo) error
}
