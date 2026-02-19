package logger

import (
	"fmt"
	"os"

	"github.com/philipp01105/nlog/core"
	"github.com/philipp01105/nlog/handler"
)

// osExit is a variable to allow overriding os.Exit in tests
var osExit = os.Exit

// Logger is the main logging interface (immutable)
type Logger struct {
	handler       handler.Handler
	level         core.Level
	fields        []core.Field
	includeCaller bool
	callerSkip    int
	recycleEntry  bool
}

// Builder provides a fluent API for building Logger instances
type Builder struct {
	handler       handler.Handler
	level         core.Level
	fields        []core.Field
	includeCaller bool
	callerSkip    int
}

// NewBuilder creates a new logger builder
func NewBuilder() *Builder {
	return &Builder{
		level:      core.InfoLevel, // Default level
		callerSkip: 3,              // Default skip for getCaller
	}
}

// WithHandler sets the handler
func (b *Builder) WithHandler(h handler.Handler) *Builder {
	b.handler = h
	return b
}

// WithLevel sets the log level
func (b *Builder) WithLevel(level core.Level) *Builder {
	b.level = level
	return b
}

// WithFields adds default fields to all log entries
func (b *Builder) WithFields(fields ...core.Field) *Builder {
	b.fields = append(b.fields, fields...)
	return b
}

// WithCaller enables caller information
func (b *Builder) WithCaller(enabled bool) *Builder {
	b.includeCaller = enabled
	return b
}

// Build creates the Logger instance
func (b *Builder) Build() *Logger {
	l := &Logger{
		handler:       b.handler,
		level:         b.level,
		fields:        b.fields,
		includeCaller: b.includeCaller,
		callerSkip:    b.callerSkip,
	}
	// Check if handler supports entry recycling
	if rc, ok := l.handler.(interface{ CanRecycleEntry() bool }); ok {
		l.recycleEntry = rc.CanRecycleEntry()
	}
	return l
}

// With creates a new Logger with additional fields (immutable operation)
func (l *Logger) With(fields ...core.Field) *Logger {
	newFields := make([]core.Field, len(l.fields)+len(fields))
	copy(newFields, l.fields)
	copy(newFields[len(l.fields):], fields)

	return &Logger{
		handler:       l.handler,
		level:         l.level,
		fields:        newFields,
		includeCaller: l.includeCaller,
		callerSkip:    l.callerSkip,
		recycleEntry:  l.recycleEntry,
	}
}

// Log logs a message at the specified level
func (l *Logger) Log(level core.Level, msg string, fields ...core.Field) {
	// Level check optimization - exit early BEFORE any allocations
	if level < l.level {
		return
	}

	l.log(level, msg, fields)
}

// log is the internal logging method that takes a pre-allocated slice
func (l *Logger) log(level core.Level, msg string, fields []core.Field) {
	// Handler check - exit if no handler (avoid any work)
	if l.handler == nil {
		return
	}

	// Get entry from pool AFTER level check
	entry := core.GetEntry()
	entry.Level = level
	entry.Message = msg

	// Add logger's default fields
	if len(l.fields) > 0 {
		entry.Fields = append(entry.Fields, l.fields...)
	}

	// Add provided fields
	if len(fields) > 0 {
		entry.Fields = append(entry.Fields, fields...)
	}

	if l.includeCaller {
		entry.Caller = core.GetCaller(l.callerSkip)
	}

	err := l.handler.Handle(entry)
	if err != nil {
		return
	}

	// Return entry to pool if handler supports it
	if l.recycleEntry {
		core.PutEntry(entry)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...core.Field) {
	if core.DebugLevel < l.level {
		return
	}
	l.log(core.DebugLevel, msg, fields)
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...core.Field) {
	if core.InfoLevel < l.level {
		return
	}
	l.log(core.InfoLevel, msg, fields)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...core.Field) {
	if core.WarnLevel < l.level {
		return
	}
	l.log(core.WarnLevel, msg, fields)
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...core.Field) {
	if core.ErrorLevel < l.level {
		return
	}
	l.log(core.ErrorLevel, msg, fields)
}

// Fatal logs a fatal message and exits the program with os.Exit(1)
func (l *Logger) Fatal(msg string, fields ...core.Field) {
	l.log(core.FatalLevel, msg, fields)
	osExit(1)
}

// Panic logs a panic message and panics
func (l *Logger) Panic(msg string, fields ...core.Field) {
	l.log(core.PanicLevel, msg, fields)
	panic(msg)
}

// Debugf logs a debug message with formatting
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Log(core.DebugLevel, fmt.Sprintf(format, args...))
}

// Infof logs an info message with formatting
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Log(core.InfoLevel, fmt.Sprintf(format, args...))
}

// Warnf logs a warning message with formatting
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Log(core.WarnLevel, fmt.Sprintf(format, args...))
}

// Errorf logs an error message with formatting
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Log(core.ErrorLevel, fmt.Sprintf(format, args...))
}

// Fatalf logs a fatal message with formatting and exits the program with os.Exit(1)
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Log(core.FatalLevel, fmt.Sprintf(format, args...))
	osExit(1)
}

// Panicf logs a panic message with formatting and panics
func (l *Logger) Panicf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.Log(core.PanicLevel, msg)
	panic(msg)
}

// Close closes the logger's handler
func (l *Logger) Close() error {
	if l.handler != nil {
		return l.handler.Close()
	}
	return nil
}
