// Package handler provides the Handler interface and common types shared
// by all handler implementations.
//
// Handler implementations are organized in sub-packages for better
// modularity, testability, and separation of concerns:
//
//   - handler/consolehandler – console output (SyncConsoleHandler,
//     AsyncConsoleHandler). Created via consolehandler.NewConsoleHandler.
//   - handler/filehandler – file output with automatic rotation
//     (SyncFileHandler, AsyncFileHandler). Created via
//     filehandler.NewFileHandler.
//   - handler/multihandler – fan-out to multiple child handlers.
//     Created via multihandler.NewMultiHandler.
//   - handler/sloghandler – adapter from Handler to log/slog.Handler.
//     Created via sloghandler.NewSlogHandler.
//
// This package defines the shared interfaces and types used across all
// sub-packages:
//
//   - Handler and FastHandler interfaces for log entry processing.
//   - StatsProvider interface for runtime statistics monitoring.
//   - OverflowPolicy (DropNewest, DropOldest, Block) for async queue
//     overflow behavior.
//   - Stats and Snapshot types for tracking dropped, blocked, and
//     processed log counts.
package handler
