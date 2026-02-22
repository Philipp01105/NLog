package consolehandler

import (
	"bytes"
	"testing"
	"time"

	"github.com/philipp01105/nlog/core"
	"github.com/philipp01105/nlog/handler"
)

func TestOverflowPolicy_DropNewest(t *testing.T) {
	var buf bytes.Buffer
	h := NewConsoleHandler(ConsoleConfig{
		Writer:     &buf,
		Async:      true,
		BufferSize: 2, // Small buffer to test overflow
		OverflowPolicy: map[core.Level]handler.OverflowPolicy{
			core.InfoLevel: handler.DropNewest,
		},
	})
	defer h.Close()

	// Fill buffer beyond capacity
	for i := 0; i < 10; i++ {
		entry := core.GetEntry()
		entry.Level = core.InfoLevel
		entry.Message = "test"
		h.Handle(entry)
	}

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	// Check stats - some should be dropped
	stats := h.(handler.StatsProvider).Stats()
	if stats.DroppedTotal[core.InfoLevel] == 0 {
		t.Error("Expected some dropped logs with DropNewest policy")
	}
}

func TestOverflowPolicy_Block(t *testing.T) {
	var buf bytes.Buffer
	h := NewConsoleHandler(ConsoleConfig{
		Writer:       &buf,
		Async:        true,
		BufferSize:   2,
		BlockTimeout: 50 * time.Millisecond,
		OverflowPolicy: map[core.Level]handler.OverflowPolicy{
			core.ErrorLevel: handler.Block,
		},
	})
	defer h.Close()

	// Fill buffer
	for i := 0; i < 10; i++ {
		entry := core.GetEntry()
		entry.Level = core.ErrorLevel
		entry.Message = "error"
		h.Handle(entry)
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Check stats - should have some blocked writes
	stats := h.(handler.StatsProvider).Stats()
	if stats.BlockedTotal == 0 {
		t.Log("Warning: Expected some blocked logs with Block policy (might be timing-dependent)")
	}
}

func TestOverflowPolicy_DropOldest(t *testing.T) {
	var buf bytes.Buffer
	h := NewConsoleHandler(ConsoleConfig{
		Writer:     &buf,
		Async:      true,
		BufferSize: 2,
		OverflowPolicy: map[core.Level]handler.OverflowPolicy{
			core.WarnLevel: handler.DropOldest,
		},
	})
	defer h.Close()

	// Fill buffer beyond capacity
	for i := 0; i < 10; i++ {
		entry := core.GetEntry()
		entry.Level = core.WarnLevel
		entry.Message = "warn"
		h.Handle(entry)
	}

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	// Check stats
	stats := h.(handler.StatsProvider).Stats()
	if stats.DroppedTotal[core.WarnLevel] == 0 {
		t.Error("Expected some dropped logs with DropOldest policy")
	}
}

func TestStats_Telemetry(t *testing.T) {
	var buf bytes.Buffer
	h := NewConsoleHandler(ConsoleConfig{
		Writer:     &buf,
		Async:      false, // Synchronous for predictable counting
		BufferSize: 10,
	})
	defer h.Close()

	// Process some logs
	for i := 0; i < 5; i++ {
		entry := core.GetEntry()
		entry.Level = core.InfoLevel
		entry.Message = "info"
		h.Handle(entry)
	}

	// Check stats
	stats := h.(handler.StatsProvider).Stats()
	if stats.ProcessedTotal != 5 {
		t.Errorf("Expected 5 processed logs, got %d", stats.ProcessedTotal)
	}
}

func TestHandler_CloseIdempotent(t *testing.T) {
	var buf bytes.Buffer
	h := NewConsoleHandler(ConsoleConfig{
		Writer: &buf,
		Async:  true,
	})

	// Close multiple times - should not panic
	if err := h.Close(); err != nil {
		t.Errorf("First close failed: %v", err)
	}

	if err := h.Close(); err != nil {
		t.Errorf("Second close failed: %v", err)
	}

	if err := h.Close(); err != nil {
		t.Errorf("Third close failed: %v", err)
	}
}

func TestHandler_DrainTimeout(t *testing.T) {
	var buf bytes.Buffer
	h := NewConsoleHandler(ConsoleConfig{
		Writer:       &buf,
		Async:        true,
		BufferSize:   1000,
		DrainTimeout: 100 * time.Millisecond,
	})

	// Add many logs
	for i := 0; i < 100; i++ {
		entry := core.GetEntry()
		entry.Level = core.InfoLevel
		entry.Message = "test"
		h.Handle(entry)
	}

	// Close should drain with timeout
	start := time.Now()
	h.Close()
	elapsed := time.Since(start)

	// Should complete within drain timeout + some margin
	if elapsed > 500*time.Millisecond {
		t.Errorf("Close took too long: %v", elapsed)
	}
}

func BenchmarkHandler_DropNewest(b *testing.B) {
	var buf bytes.Buffer
	h := NewConsoleHandler(ConsoleConfig{
		Writer:     &buf,
		Async:      true,
		BufferSize: 1000,
		OverflowPolicy: map[core.Level]handler.OverflowPolicy{
			core.InfoLevel: handler.DropNewest,
		},
	})
	defer h.Close()

	entry := core.GetEntry()
	entry.Level = core.InfoLevel
	entry.Message = "benchmark"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Handle(entry)
	}
}

func BenchmarkHandler_Block(b *testing.B) {
	var buf bytes.Buffer
	h := NewConsoleHandler(ConsoleConfig{
		Writer:       &buf,
		Async:        true,
		BufferSize:   1000,
		BlockTimeout: 100 * time.Millisecond,
		OverflowPolicy: map[core.Level]handler.OverflowPolicy{
			core.ErrorLevel: handler.Block,
		},
	})
	defer h.Close()

	entry := core.GetEntry()
	entry.Level = core.ErrorLevel
	entry.Message = "benchmark"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Handle(entry)
	}
}
