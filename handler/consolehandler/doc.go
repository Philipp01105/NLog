// Package consolehandler provides console output handlers that write
// formatted log entries to any io.Writer (default: os.Stdout).
//
// Handlers are split into specialized sync and async variants:
//
//   - SyncConsoleHandler eliminates async queue overhead for a leaner
//     hot path. Uses TryLock for zero-alloc parallel formatting.
//   - AsyncConsoleHandler provides an isolated queue with per-level
//     OverflowPolicy and a dedicated background goroutine.
//
// The factory function NewConsoleHandler automatically chooses the
// right variant based on the Async field in ConsoleConfig.
package consolehandler
