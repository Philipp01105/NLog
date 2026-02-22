package consolehandler

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/philipp01105/nlog/core"
	"github.com/philipp01105/nlog/formatter"
	"github.com/philipp01105/nlog/handler"
)

func TestConsoleHandler_Sync(t *testing.T) {
	var buf bytes.Buffer
	h := NewConsoleHandler(ConsoleConfig{
		Writer:    &buf,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})
	defer h.Close()

	entry := core.GetEntry()
	entry.Level = core.InfoLevel
	entry.Message = "test message"

	err := h.Handle(entry)
	if err != nil {
		t.Errorf("Handle() error = %v", err)
	}

	if !strings.Contains(buf.String(), "test message") {
		t.Errorf("Expected 'test message' in output, got: %s", buf.String())
	}
}

func TestConsoleHandler_Async(t *testing.T) {
	var buf bytes.Buffer
	h := NewConsoleHandler(ConsoleConfig{
		Writer:     &buf,
		Async:      true,
		BufferSize: 10,
		Formatter:  formatter.NewTextFormatter(formatter.Config{}),
	})
	defer h.Close()

	entry := core.GetEntry()
	entry.Level = core.InfoLevel
	entry.Message = "async test"

	err := h.Handle(entry)
	if err != nil {
		t.Errorf("Handle() error = %v", err)
	}

	// Wait for async processing
	time.Sleep(10 * time.Millisecond)

	if !strings.Contains(buf.String(), "async test") {
		t.Errorf("Expected 'async test' in output, got: %s", buf.String())
	}
}

func TestConsoleHandler_AsyncHandleLog(t *testing.T) {
	var buf bytes.Buffer
	h := NewConsoleHandler(ConsoleConfig{
		Writer:     &buf,
		Async:      true,
		BufferSize: 100,
		Formatter:  formatter.NewTextFormatter(formatter.Config{}),
	})

	fh := h.(handler.FastHandler)
	// Use HandleLog (FastHandler path) which was previously recycling entries too early
	for i := 0; i < 50; i++ {
		fh.HandleLog(time.Now(), core.InfoLevel, "handlelog async test", nil, nil, core.CallerInfo{})
	}

	h.Close()

	output := buf.String()
	if !strings.Contains(output, "handlelog async test") {
		t.Errorf("Expected 'handlelog async test' in output, got: %s", output)
	}
	count := strings.Count(output, "handlelog async test")
	if count != 50 {
		t.Errorf("Expected 50 messages, got %d", count)
	}
}

func TestConsoleHandler_DropNewest(t *testing.T) {
	var buf bytes.Buffer
	h := NewConsoleHandler(ConsoleConfig{
		Writer:     &buf,
		Async:      true,
		BufferSize: 2, // Small buffer to test drop
		Formatter:  formatter.NewTextFormatter(formatter.Config{}),
	})
	defer h.Close()

	// Fill the buffer beyond capacity
	for i := 0; i < 10; i++ {
		entry := core.GetEntry()
		entry.Level = core.InfoLevel
		entry.Message = "test"
		h.Handle(entry)
	}

	// Should not block even though buffer is full
	time.Sleep(10 * time.Millisecond)
}

func TestIsConcurrentSafeWriter(t *testing.T) {
	tests := []struct {
		name     string
		writer   io.Writer
		expected bool
	}{
		{"io.Discard", io.Discard, true},
		{"os.Stdout", os.Stdout, true},
		{"os.Stderr", os.Stderr, true},
		{"bytes.Buffer", &bytes.Buffer{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isConcurrentSafeWriter(tt.writer); got != tt.expected {
				t.Errorf("isConcurrentSafeWriter(%T) = %v, want %v", tt.writer, got, tt.expected)
			}
		})
	}
}

func TestConcurrentSafeConfig(t *testing.T) {
	// Auto-detected for io.Discard
	h := NewConsoleHandler(ConsoleConfig{
		Writer:    io.Discard,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})
	if !h.(*SyncConsoleHandler).concurrentSafe {
		t.Error("Expected concurrentSafe=true for io.Discard")
	}
	h.Close()

	// Not auto-detected for bytes.Buffer
	h = NewConsoleHandler(ConsoleConfig{
		Writer:    &bytes.Buffer{},
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})
	if h.(*SyncConsoleHandler).concurrentSafe {
		t.Error("Expected concurrentSafe=false for bytes.Buffer")
	}
	h.Close()

	// Explicit opt-in via ConcurrentWriter
	h = NewConsoleHandler(ConsoleConfig{
		Writer:           &bytes.Buffer{},
		Async:            false,
		Formatter:        formatter.NewTextFormatter(formatter.Config{}),
		ConcurrentWriter: true,
	})
	if !h.(*SyncConsoleHandler).concurrentSafe {
		t.Error("Expected concurrentSafe=true with ConcurrentWriter=true")
	}
	h.Close()
}

func TestConsoleHandler_ConcurrentSafe_Parallel(t *testing.T) {
	h := NewConsoleHandler(ConsoleConfig{
		Writer:    io.Discard,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})
	defer h.Close()

	fh := h.(handler.FastHandler)

	const goroutines = 8
	const msgs = 100
	done := make(chan struct{}, goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for i := 0; i < msgs; i++ {
				fh.HandleLog(time.Now(), core.InfoLevel, "parallel safe test", nil, nil, core.CallerInfo{})
			}
		}()
	}
	for g := 0; g < goroutines; g++ {
		<-done
	}

	snap := h.(handler.StatsProvider).Stats()
	if snap.ProcessedTotal != goroutines*msgs {
		t.Errorf("Expected %d processed, got %d", goroutines*msgs, snap.ProcessedTotal)
	}
}
