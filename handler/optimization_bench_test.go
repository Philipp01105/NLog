package handler

import (
	"io"
	"os"
	"testing"

	"github.com/philipp01105/nlog/core"
	"github.com/philipp01105/nlog/formatter"
)

// BenchmarkFileHandler_WriterFormatter benchmarks FileHandler with WriterFormatter path (H1.1 + H1.2)
func BenchmarkFileHandler_WriterFormatter(b *testing.B) {
	dir := b.TempDir()
	filename := dir + "/bench.log"

	h, err := NewFileHandler(FileConfig{
		Filename:  filename,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})
	if err != nil {
		b.Fatal(err)
	}
	defer h.Close()

	entry := core.GetEntry()
	entry.Level = core.InfoLevel
	entry.Message = "benchmark message"
	entry.Fields = []core.Field{
		{Key: "key1", Type: core.StringType, Str: "value1"},
		{Key: "key2", Type: core.Int64Type, Int64: 42},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		h.write(entry)
	}
}

// BenchmarkFileHandler_JSONFormatter benchmarks FileHandler with JSON formatter (H1.1)
func BenchmarkFileHandler_JSONFormatter(b *testing.B) {
	dir := b.TempDir()
	filename := dir + "/bench.log"

	h, err := NewFileHandler(FileConfig{
		Filename:  filename,
		Async:     false,
		Formatter: formatter.NewJSONFormatter(formatter.Config{}),
	})
	if err != nil {
		b.Fatal(err)
	}
	defer h.Close()

	entry := core.GetEntry()
	entry.Level = core.InfoLevel
	entry.Message = "benchmark message"
	entry.Fields = []core.Field{
		{Key: "key1", Type: core.StringType, Str: "value1"},
		{Key: "key2", Type: core.Int64Type, Int64: 42},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		h.write(entry)
	}
}

// BenchmarkConsoleHandler_WriterFormatter benchmarks ConsoleHandler with WriterFormatter (zero-alloc path)
func BenchmarkConsoleHandler_WriterFormatter(b *testing.B) {
	h := NewConsoleHandler(ConsoleConfig{
		Writer:    io.Discard,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})
	defer h.Close()

	entry := core.GetEntry()
	entry.Level = core.InfoLevel
	entry.Message = "benchmark message"
	entry.Fields = []core.Field{
		{Key: "key1", Type: core.StringType, Str: "value1"},
		{Key: "key2", Type: core.Int64Type, Int64: 42},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		h.write(entry)
	}
}

// BenchmarkFileHandler_AsyncThroughput benchmarks FileHandler async throughput
func BenchmarkFileHandler_AsyncThroughput(b *testing.B) {
	dir := b.TempDir()
	filename := dir + "/bench.log"

	h, err := NewFileHandler(FileConfig{
		Filename:   filename,
		Async:      true,
		BufferSize: 10000,
		Formatter:  formatter.NewTextFormatter(formatter.Config{}),
	})
	if err != nil {
		b.Fatal(err)
	}
	defer h.Close()

	entry := core.GetEntry()
	entry.Level = core.InfoLevel
	entry.Message = "async throughput test"
	entry.Fields = []core.Field{
		{Key: "key1", Type: core.StringType, Str: "value1"},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		h.Handle(entry)
	}
}

// BenchmarkConsoleHandler_AsyncBatch benchmarks the batch-write path of the async ConsoleHandler.
// Multiple entries are enqueued before the consumer wakes up, triggering batch accumulation.
func BenchmarkConsoleHandler_AsyncBatch(b *testing.B) {
	h := NewConsoleHandler(ConsoleConfig{
		Writer:     io.Discard,
		Async:      true,
		BufferSize: 10000,
		Formatter:  formatter.NewTextFormatter(formatter.Config{}),
	})
	defer h.Close()

	entry := core.GetEntry()
	entry.Level = core.InfoLevel
	entry.Message = "async batch benchmark"
	entry.Fields = []core.Field{
		{Key: "key1", Type: core.StringType, Str: "value1"},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		h.Handle(entry)
	}
}

// BenchmarkFileHandler_Rotation benchmarks FileHandler with rotation enabled
func BenchmarkFileHandler_Rotation(b *testing.B) {
	dir := b.TempDir()
	filename := dir + "/bench.log"

	h, err := NewFileHandler(FileConfig{
		Filename:   filename,
		Async:      false,
		MaxSize:    1024 * 1024, // 1MB
		MaxBackups: 3,
		Formatter:  formatter.NewTextFormatter(formatter.Config{}),
	})
	if err != nil {
		b.Fatal(err)
	}
	defer h.Close()

	entry := core.GetEntry()
	entry.Level = core.InfoLevel
	entry.Message = "rotation benchmark test"

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		h.write(entry)
	}
}

// TestFileHandler_WriterFormatterPath tests that FileHandler uses WriterFormatter correctly
func TestFileHandler_WriterFormatterPath(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/test.log"

	h, err := NewFileHandler(FileConfig{
		Filename:  filename,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify that writerFormatter is set
	if h.writerFormatter == nil {
		t.Fatal("Expected writerFormatter to be set for TextFormatter")
	}

	entry := core.GetEntry()
	entry.Level = core.InfoLevel
	entry.Message = "test message"
	entry.Fields = []core.Field{
		{Key: "key1", Type: core.StringType, Str: "value1"},
	}

	if err := h.Handle(entry); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	h.Close()

	// Verify file contents
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if len(content) == 0 {
		t.Error("Expected non-empty file content")
	}
}

// TestFileHandler_BufferedIO tests that bufio.Writer is properly used
func TestFileHandler_BufferedIO(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/test.log"

	h, err := NewFileHandler(FileConfig{
		Filename:  filename,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify bufWriter is set
	if h.bufWriter == nil {
		t.Fatal("Expected bufWriter to be set")
	}

	// Write a message
	entry := core.GetEntry()
	entry.Level = core.InfoLevel
	entry.Message = "buffered IO test"
	if err := h.Handle(entry); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	// Close should flush the buffer
	if err := h.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Verify file was written
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("Expected file to contain data after close+flush")
	}
}
