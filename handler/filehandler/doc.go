// Package filehandler provides file output handlers that write formatted
// log entries to files with automatic rotation by size, age, or interval.
//
// Handlers are split into specialized sync and async variants:
//
//   - SyncFileHandler eliminates async queue overhead for the hot path.
//   - AsyncFileHandler provides an isolated queue with per-level
//     OverflowPolicy and a dedicated background goroutine.
//
// The factory function NewFileHandler automatically chooses the right
// variant based on the Async field in FileConfig.
package filehandler
