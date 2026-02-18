package logger

import (
	"io"
	"testing"

	"github.com/Philipp01105/logging-framework/formatter"
	"github.com/Philipp01105/logging-framework/handler"
)

// BenchmarkInfoNoFields benchmarks Info() with no fields using a discard writer.
// Target: <150 ns/op, 0 allocs/op, 0 B/op
func BenchmarkInfoNoFields(b *testing.B) {
	h := handler.NewConsoleHandler(handler.ConsoleConfig{
		Writer:    io.Discard,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})
	defer h.Close()

	logger := NewBuilder().
		WithHandler(h).
		WithLevel(InfoLevel).
		Build()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger.Info("test message")
	}
}

// BenchmarkInfoWith2Fields benchmarks Info() with 2 string fields using a discard writer.
// Target: <250 ns/op, 0-1 allocs/op, <128 B/op
func BenchmarkInfoWith2Fields(b *testing.B) {
	h := handler.NewConsoleHandler(handler.ConsoleConfig{
		Writer:    io.Discard,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})
	defer h.Close()

	logger := NewBuilder().
		WithHandler(h).
		WithLevel(InfoLevel).
		Build()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger.Info("test message", String("key1", "value1"), String("key2", "value2"))
	}
}

// BenchmarkFilteredDebug benchmarks Debug() when level is Info (should be filtered).
// Target: <10 ns/op, 0 allocs/op, 0 B/op
func BenchmarkFilteredDebug(b *testing.B) {
	h := handler.NewConsoleHandler(handler.ConsoleConfig{
		Writer:    io.Discard,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})
	defer h.Close()

	logger := NewBuilder().
		WithHandler(h).
		WithLevel(InfoLevel).
		Build()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger.Debug("debug message", String("key", "value"))
	}
}

// BenchmarkJSON benchmarks Info() with JSON formatter.
// Target: <500 ns/op, 0-1 allocs/op, <256 B/op
func BenchmarkJSON(b *testing.B) {
	h := handler.NewConsoleHandler(handler.ConsoleConfig{
		Writer:    io.Discard,
		Async:     false,
		Formatter: formatter.NewJSONFormatter(formatter.Config{}),
	})
	defer h.Close()

	logger := NewBuilder().
		WithHandler(h).
		WithLevel(InfoLevel).
		Build()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger.Info("test message", String("key1", "value1"), String("key2", "value2"))
	}
}
