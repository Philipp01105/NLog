package logger

import (
	"sync"

	"github.com/philipp01105/nlog/core"
	"github.com/philipp01105/nlog/formatter"
	"github.com/philipp01105/nlog/handler/consolehandler"
)

var (
	defaultLogger *Logger
	defaultMu     sync.RWMutex
)

func init() {
	// Initialize default logger with console handler
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Async:      true,
		BufferSize: 1000,
		Formatter:  formatter.NewTextFormatter(formatter.Config{}),
	})

	defaultLogger = NewBuilder().
		WithHandler(h).
		WithLevel(core.InfoLevel).
		Build()
}

// Default returns the default logger
func Default() *Logger {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultLogger
}

// SetDefault sets the default logger
func SetDefault(l *Logger) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	defaultLogger = l
}

// Package-level convenience functions using the default logger

// Debug logs a debug message using the default logger
func Debug(msg string, fields ...core.Field) {
	Default().Debug(msg, fields...)
}

// Info logs an info message using the default logger
func Info(msg string, fields ...core.Field) {
	Default().Info(msg, fields...)
}

// Warn logs a warning message using the default logger
func Warn(msg string, fields ...core.Field) {
	Default().Warn(msg, fields...)
}

// Error logs an error message using the default logger
func Error(msg string, fields ...core.Field) {
	Default().Error(msg, fields...)
}

// Fatal logs a fatal message using the default logger and exits the program
func Fatal(msg string, fields ...core.Field) {
	Default().Fatal(msg, fields...)
}

// Panic logs a panic message using the default logger and panics
func Panic(msg string, fields ...core.Field) {
	Default().Panic(msg, fields...)
}

// Debugf logs a formatted debug message using the default logger
func Debugf(format string, args ...interface{}) {
	Default().Debugf(format, args...)
}

// Infof logs a formatted info message using the default logger
func Infof(format string, args ...interface{}) {
	Default().Infof(format, args...)
}

// Warnf logs a formatted warning message using the default logger
func Warnf(format string, args ...interface{}) {
	Default().Warnf(format, args...)
}

// Errorf logs a formatted error message using the default logger
func Errorf(format string, args ...interface{}) {
	Default().Errorf(format, args...)
}

// Fatalf logs a formatted fatal message using the default logger and exits the program
func Fatalf(format string, args ...interface{}) {
	Default().Fatalf(format, args...)
}

// Panicf logs a formatted panic message using the default logger and panics
func Panicf(format string, args ...interface{}) {
	Default().Panicf(format, args...)
}

// With creates a new logger with additional fields
func With(fields ...core.Field) *Logger {
	return Default().With(fields...)
}
