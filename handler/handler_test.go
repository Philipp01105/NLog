package handler

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/Philipp01105/logging-framework/core"
	"github.com/Philipp01105/logging-framework/formatter"
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

func TestMultiHandler(t *testing.T) {
	var buf1, buf2 bytes.Buffer

	h1 := NewConsoleHandler(ConsoleConfig{
		Writer:    &buf1,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})

	h2 := NewConsoleHandler(ConsoleConfig{
		Writer:    &buf2,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})

	multi := NewMultiHandler(h1, h2)
	defer multi.Close()

	entry := core.GetEntry()
	entry.Level = core.InfoLevel
	entry.Message = "multi test"

	err := multi.Handle(entry)
	if err != nil {
		t.Errorf("Handle() error = %v", err)
	}

	if !strings.Contains(buf1.String(), "multi test") {
		t.Error("First handler did not receive message")
	}

	if !strings.Contains(buf2.String(), "multi test") {
		t.Error("Second handler did not receive message")
	}
}
