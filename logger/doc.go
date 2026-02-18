// Package logger is the public API of NLog. Most users only need to
// import this package.
//
// A Logger is immutable after construction — all fields, the level,
// and the handler are set once via the Builder and never modified.
// This makes Logger inherently safe for concurrent use without any
// locking on the read path.
//
// The package initializes a default Logger (async, InfoLevel, text
// format to stdout) in init(). The package-level functions Info,
// Error, Debugf, etc. delegate to this default instance, so simple
// programs can log without any setup:
//
//	logger.Info("ready", logger.Int("port", 8080))
//
// For custom configuration, use the Builder:
//
//	log := logger.NewBuilder().
//	    WithHandler(myHandler).
//	    WithLevel(logger.DebugLevel).
//	    WithCaller(true).
//	    Build()
//
// Child loggers with extra fields are created via With, which returns
// a new Logger that shares the same handler but carries additional
// default fields:
//
//	reqLog := log.With(logger.String("request_id", id))
//
// Level checks happen before any allocation, so filtered-out
// messages cost only a single integer comparison.
package logger
