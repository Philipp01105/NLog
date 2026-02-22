// Package sloghandler provides an adapter from handler.Handler to
// log/slog.Handler, allowing nlog to serve as a drop-in backend for
// the standard library's structured logging package.
//
// Create an adapter with NewSlogHandler and pass it to slog.New:
//
//	sh := sloghandler.NewSlogHandler(myHandler, core.InfoLevel)
//	logger := slog.New(sh)
//	logger.Info("hello", "key", "value")
//
// The adapter converts slog.Record attributes to nlog fields and
// forwards them to the underlying handler.
package sloghandler
