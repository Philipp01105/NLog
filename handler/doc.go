// Package handler provides the Handler interface and its built-in
// implementations for dispatching log entries to various outputs.
//
// All handlers support both synchronous and asynchronous operations.
// In async mode, entries are sent to a bounded channel and processed
// by a background goroutine, which keeps the caller's hot path fast
// even under slow I/O.
//
// When the async queue is full, each handler applies a per-level
// OverflowPolicy: DropNewest (default for Debug/Info/Warn), DropOldest,
// or Block with a configurable timeout (default for Error). This ensures
// that low-priority logs never stall the application while critical
// errors are never silently dropped.
//
// Built-in handlers:
//
//   - ConsoleHandler writes formatted entries to any io.Writer (default: stdout).
//   - FileHandler writes to a file with automatic rotation by size, age,
//     or interval, and manages old backup cleanup.
//   - MultiHandler fans out a single entry to multiple child handlers.
//   - SlogHandler adapts the Handler interface to log/slog.Handler,
//     allowing NLog to serve as a drop-in backend for the standard library.
//
// All handlers track dropped, blocked, and processed counts via the
// Stats type, which can be queried at runtime for monitoring.
package handler
