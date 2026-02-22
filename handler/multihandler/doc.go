// Package multihandler provides a fan-out handler that dispatches log
// entries to multiple child handlers. When all children implement
// FastHandler, entry allocation is avoided entirely.
package multihandler
