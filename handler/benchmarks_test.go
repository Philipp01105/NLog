package handler

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/Philipp01105/NLog/core"
	"github.com/Philipp01105/NLog/formatter"
)

// slowWriter simulates slow disk I/O
type slowWriter struct {
	delay time.Duration
	mu    sync.Mutex
}

func (w *slowWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	time.Sleep(w.delay)
	return len(p), nil
}

// BenchmarkMultiGoroutineContention tests handler under concurrent load
func BenchmarkMultiGoroutineContention(b *testing.B) {
	var buf bytes.Buffer
	h := NewConsoleHandler(ConsoleConfig{
		Writer:     &buf,
		Async:      true,
		BufferSize: 10000,
	})
	defer h.Close()

	entry := core.GetEntry()
	entry.Level = core.InfoLevel
	entry.Message = "concurrent log"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			h.Handle(entry)
		}
	})
}

// BenchmarkQueueFullStress tests handler behavior when queue is constantly full
func BenchmarkQueueFullStress(b *testing.B) {
	// Use slow writer to keep queue full
	sw := &slowWriter{delay: 10 * time.Millisecond}
	h := NewConsoleHandler(ConsoleConfig{
		Writer:     sw,
		Async:      true,
		BufferSize: 10, // Small buffer
		OverflowPolicy: map[core.Level]OverflowPolicy{
			core.InfoLevel: DropNewest,
		},
	})
	defer h.Close()

	entry := core.GetEntry()
	entry.Level = core.InfoLevel
	entry.Message = "stress test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Handle(entry)
	}
	b.StopTimer()

	stats := h.Stats()
	b.ReportMetric(float64(stats.DroppedTotal[core.InfoLevel]), "dropped")
}

// BenchmarkSlowDiskSimulation tests handler with simulated slow disk
func BenchmarkSlowDiskSimulation(b *testing.B) {
	sw := &slowWriter{delay: 1 * time.Millisecond}
	h := NewConsoleHandler(ConsoleConfig{
		Writer:     sw,
		Async:      true,
		BufferSize: 1000,
		OverflowPolicy: map[core.Level]OverflowPolicy{
			core.ErrorLevel: Block,
		},
		BlockTimeout: 50 * time.Millisecond,
	})
	defer h.Close()

	entry := core.GetEntry()
	entry.Level = core.ErrorLevel
	entry.Message = "slow disk test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Handle(entry)
	}
	b.StopTimer()

	stats := h.Stats()
	b.ReportMetric(float64(stats.BlockedTotal), "blocked")
}

// BenchmarkMemoryUnderLoad measures memory behavior under sustained load
func BenchmarkMemoryUnderLoad(b *testing.B) {
	var buf bytes.Buffer
	h := NewConsoleHandler(ConsoleConfig{
		Writer:     &buf,
		Async:      true,
		BufferSize: 1000,
	})
	defer h.Close()

	entry := core.GetEntry()
	entry.Level = core.InfoLevel
	entry.Message = "memory test message with some content to measure allocation"
	entry.Fields = []core.Field{
		{Key: "key1", Type: core.StringType, Str: "value1"},
		{Key: "key2", Type: core.Int64Type, Int64: 42},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create new entry each time to measure allocation
		e := core.GetEntry()
		e.Level = entry.Level
		e.Message = entry.Message
		e.Fields = append(e.Fields, entry.Fields...)
		h.Handle(e)
	}
}

// TestQueueBehaviorObservable tests that we can observe queue behavior through stats
func TestQueueBehaviorObservable(t *testing.T) {
	var buf bytes.Buffer
	h := NewConsoleHandler(ConsoleConfig{
		Writer:     &buf,
		Async:      true,
		BufferSize: 10,
		OverflowPolicy: map[core.Level]OverflowPolicy{
			core.InfoLevel: DropNewest,
		},
	})

	// Send more than buffer
	for i := 0; i < 50; i++ {
		entry := core.GetEntry()
		entry.Level = core.InfoLevel
		entry.Message = "test"
		h.Handle(entry)
	}

	// Close and wait
	h.Close()

	// Check stats - should have processed some and dropped some
	stats := h.Stats()
	total := stats.ProcessedTotal + stats.DroppedTotal[core.InfoLevel]
	if total != 50 {
		t.Errorf("Expected 50 total (processed+dropped), got %d", total)
	}
	if stats.DroppedTotal[core.InfoLevel] == 0 {
		t.Error("Expected some dropped logs")
	}

	t.Logf("Processed: %d, Dropped: %d", stats.ProcessedTotal, stats.DroppedTotal[core.InfoLevel])
}

// TestMemoryBounded verifies that queue doesn't grow unbounded
func TestMemoryBounded(t *testing.T) {
	var buf bytes.Buffer
	h := NewConsoleHandler(ConsoleConfig{
		Writer:     &buf,
		Async:      true,
		BufferSize: 10, // Small bounded queue
		OverflowPolicy: map[core.Level]OverflowPolicy{
			core.InfoLevel: DropNewest,
		},
	})

	// Try to send way more than buffer can hold
	for i := 0; i < 100; i++ {
		entry := core.GetEntry()
		entry.Level = core.InfoLevel
		entry.Message = "test"
		h.Handle(entry)
	}

	// Close will drain with timeout
	h.Close()

	// Check that many were dropped (memory was bounded)
	stats := h.Stats()
	if stats.DroppedTotal[core.InfoLevel] == 0 {
		t.Error("Expected dropped logs due to bounded queue")
	}

	t.Logf("Dropped: %d, Processed: %d", stats.DroppedTotal[core.InfoLevel], stats.ProcessedTotal)
}

// BenchmarkDifferentPolicies compares different overflow policies
func BenchmarkDifferentPolicies(b *testing.B) {
	policies := []struct {
		name   string
		policy OverflowPolicy
	}{
		{"DropNewest", DropNewest},
		{"DropOldest", DropOldest},
		{"Block", Block},
	}

	for _, p := range policies {
		b.Run(p.name, func(b *testing.B) {
			var buf bytes.Buffer
			h := NewConsoleHandler(ConsoleConfig{
				Writer:     &buf,
				Async:      true,
				BufferSize: 100,
				OverflowPolicy: map[core.Level]OverflowPolicy{
					core.InfoLevel: p.policy,
				},
				BlockTimeout: 10 * time.Millisecond,
			})
			defer h.Close()

			entry := core.GetEntry()
			entry.Level = core.InfoLevel
			entry.Message = "benchmark"

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				h.Handle(entry)
			}
		})
	}
}

// TestTelemetryAccuracy verifies telemetry counts are accurate
func TestTelemetryAccuracy(t *testing.T) {
	var buf bytes.Buffer
	h := NewConsoleHandler(ConsoleConfig{
		Writer:     &buf,
		Async:      false, // Synchronous for exact counting
		BufferSize: 10,
	})
	defer h.Close()

	// Process exactly 10 logs
	for i := 0; i < 10; i++ {
		entry := core.GetEntry()
		entry.Level = core.InfoLevel
		entry.Message = "test"
		h.Handle(entry)
	}

	stats := h.Stats()
	if stats.ProcessedTotal != 10 {
		t.Errorf("Expected 10 processed, got %d", stats.ProcessedTotal)
	}
	if stats.DroppedTotal[core.InfoLevel] != 0 {
		t.Errorf("Expected 0 dropped, got %d", stats.DroppedTotal[core.InfoLevel])
	}
}

// TestConcurrentStats verifies stats are thread-safe
func TestConcurrentStats(t *testing.T) {
	var buf bytes.Buffer
	h := NewConsoleHandler(ConsoleConfig{
		Writer:     &buf,
		Async:      true,
		BufferSize: 1000,
	})
	defer h.Close()

	const numGoroutines = 10
	const logsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent logging
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < logsPerGoroutine; j++ {
				entry := core.GetEntry()
				entry.Level = core.InfoLevel
				entry.Message = "concurrent"
				h.Handle(entry)
			}
		}()
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond) // Wait for processing

	// Stats should be thread-safe to read
	stats := h.Stats()
	expectedMin := uint64(numGoroutines * logsPerGoroutine)
	total := stats.ProcessedTotal + stats.DroppedTotal[core.InfoLevel]

	if total < expectedMin {
		t.Errorf("Expected at least %d total (processed+dropped), got %d", expectedMin, total)
	}
}

// discard Writer is an io.Writer that discards all data
type discardWriter struct{}

func (d *discardWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// BenchmarkHighThroughput tests maximum throughput
func BenchmarkHighThroughput(b *testing.B) {
	h := NewConsoleHandler(ConsoleConfig{
		Writer:     &discardWriter{},
		Async:      true,
		BufferSize: 10000,
		Formatter:  formatter.NewTextFormatter(formatter.Config{}),
	})
	defer h.Close()

	entry := core.GetEntry()
	entry.Level = core.InfoLevel
	entry.Message = "high throughput test"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			h.Handle(entry)
		}
	})
	b.StopTimer()

	stats := h.Stats()
	b.ReportMetric(float64(stats.ProcessedTotal), "processed")
	b.ReportMetric(float64(stats.DroppedTotal[core.InfoLevel]), "dropped")
}
