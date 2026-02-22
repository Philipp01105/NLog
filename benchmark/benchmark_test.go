package benchmark

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/philipp01105/nlog/core"
	"github.com/philipp01105/nlog/formatter"
	"github.com/philipp01105/nlog/handler"
	"github.com/philipp01105/nlog/handler/consolehandler"
	"github.com/philipp01105/nlog/handler/filehandler"
	"github.com/philipp01105/nlog/handler/multihandler"
	"github.com/philipp01105/nlog/logger"
)

// discardWriter is a no-op writer for benchmarking
type discardWriter struct{}

func (w discardWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

var (
	sinkBytes []byte
	sinkField any
	sinkU64   uint64
)

// Benchmark logger creation
func BenchmarkLoggerCreation(b *testing.B) {
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    discardWriter{},
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
		Async:     false,
	})
	defer h.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = logger.NewBuilder().
			WithHandler(h).
			WithLevel(core.InfoLevel).
			Build()
	}
}

// Benchmark logger creation with fields
func BenchmarkLoggerCreationWithFields(b *testing.B) {
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    discardWriter{},
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
		Async:     false,
	})
	defer h.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = logger.NewBuilder().
			WithHandler(h).
			WithLevel(core.InfoLevel).
			WithFields(
				logger.String("service", "test"),
				logger.String("version", "1.0.0"),
			).
			Build()
	}
}

// Benchmark With() method (creating child loggers)
func BenchmarkWith(b *testing.B) {
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    discardWriter{},
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
		Async:     false,
	})
	defer h.Close()

	log := logger.NewBuilder().
		WithHandler(h).
		WithLevel(core.InfoLevel).
		Build()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = log.With(logger.String("request_id", "12345"))
	}
}

// Benchmark basic Info logging without fields
func BenchmarkInfoNoFields(b *testing.B) {
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    discardWriter{},
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
		Async:     false,
	})
	defer h.Close()

	log := logger.NewBuilder().
		WithHandler(h).
		WithLevel(core.InfoLevel).
		Build()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.Info("test message")
	}
}

// Benchmark Info logging with 1 field
func BenchmarkInfo1Field(b *testing.B) {
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    discardWriter{},
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
		Async:     false,
	})
	defer h.Close()

	log := logger.NewBuilder().
		WithHandler(h).
		WithLevel(core.InfoLevel).
		Build()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.Info("test message", logger.String("key", "value"))
	}
}

// Benchmark Info logging with 5 fields
func BenchmarkInfo5Fields(b *testing.B) {
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    discardWriter{},
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
		Async:     false,
	})
	defer h.Close()

	log := logger.NewBuilder().
		WithHandler(h).
		WithLevel(core.InfoLevel).
		Build()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.Info("test message",
			logger.String("key1", "value1"),
			logger.Int("key2", 42),
			logger.Float64("key3", 3.14),
			logger.Bool("key4", true),
			logger.String("key5", "value5"),
		)
	}
}

// Benchmark Info logging with 10 fields
func BenchmarkInfo10Fields(b *testing.B) {
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    discardWriter{},
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
		Async:     false,
	})
	defer h.Close()

	log := logger.NewBuilder().
		WithHandler(h).
		WithLevel(core.InfoLevel).
		Build()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.Info("test message",
			logger.String("key1", "value1"),
			logger.Int("key2", 42),
			logger.Float64("key3", 3.14),
			logger.Bool("key4", true),
			logger.String("key5", "value5"),
			logger.Int64("key6", 1234567890),
			logger.Duration("key7", time.Second),
			logger.Time("key8", time.Now()),
			logger.String("key9", "value9"),
			logger.String("key10", "value10"),
		)
	}
}

// Benchmark disabled level (testing early exit optimization)
func BenchmarkDisabledLevel(b *testing.B) {
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    discardWriter{},
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
		Async:     false,
	})
	defer h.Close()

	log := logger.NewBuilder().
		WithHandler(h).
		WithLevel(core.ErrorLevel). // Only errors and above
		Build()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.Debug("debug message", logger.String("key", "value"))
	}
}

// Benchmark different field types
func BenchmarkFieldTypes(b *testing.B) {
	tests := []struct {
		name  string
		field core.Field
	}{
		{"String", logger.String("key", "value")},
		{"Int", logger.Int("key", 42)},
		{"Int64", logger.Int64("key", 1234567890)},
		{"Float64", logger.Float64("key", 3.14159265)},
		{"Bool", logger.Bool("key", true)},
		{"Time", logger.Time("key", time.Now())},
		{"Duration", logger.Duration("key", time.Second)},
		{"Error", logger.Err(errors.New("test error"))},
		{"Any", logger.Any("key", map[string]string{"nested": "value"})},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
				Writer:    discardWriter{},
				Formatter: formatter.NewTextFormatter(formatter.Config{}),
				Async:     false,
			})
			defer h.Close()

			log := logger.NewBuilder().
				WithHandler(h).
				WithLevel(core.InfoLevel).
				Build()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				log.Info("test message", tt.field)
			}
		})
	}
}

// Benchmark Text vs JSON formatter
func BenchmarkFormatters(b *testing.B) {
	tests := []struct {
		name      string
		formatter formatter.Formatter
	}{
		{"Text", formatter.NewTextFormatter(formatter.Config{})},
		{"JSON", formatter.NewJSONFormatter(formatter.Config{})},
		{"TextWithCaller", formatter.NewTextFormatter(formatter.Config{IncludeCaller: true})},
		{"JSONWithCaller", formatter.NewJSONFormatter(formatter.Config{IncludeCaller: true})},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
				Writer:    discardWriter{},
				Formatter: tt.formatter,
				Async:     false,
			})
			defer h.Close()

			log := logger.NewBuilder().
				WithHandler(h).
				WithLevel(core.InfoLevel).
				Build()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				log.Info("test message",
					logger.String("key1", "value1"),
					logger.Int("key2", 42),
					logger.Float64("key3", 3.14),
				)
			}
		})
	}
}

// Benchmark sync vs async handler
func BenchmarkSyncVsAsync(b *testing.B) {
	tests := []struct {
		name  string
		async bool
	}{
		{"Sync", false},
		{"Async", true},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
				Writer:     discardWriter{},
				Formatter:  formatter.NewTextFormatter(formatter.Config{}),
				Async:      tt.async,
				BufferSize: 10000,
			})
			defer h.Close()

			log := logger.NewBuilder().
				WithHandler(h).
				WithLevel(core.InfoLevel).
				Build()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				log.Info("test message",
					logger.String("key1", "value1"),
					logger.Int("key2", i),
				)
			}
		})
	}
}

// Benchmark logging with caller info
func BenchmarkWithCaller(b *testing.B) {
	tests := []struct {
		name          string
		includeCaller bool
	}{
		{"WithoutCaller", false},
		{"WithCaller", true},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
				Writer:    discardWriter{},
				Formatter: formatter.NewTextFormatter(formatter.Config{IncludeCaller: tt.includeCaller}),
				Async:     false,
			})
			defer h.Close()

			log := logger.NewBuilder().
				WithHandler(h).
				WithLevel(core.InfoLevel).
				WithCaller(tt.includeCaller).
				Build()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				log.Info("test message", logger.String("key", "value"))
			}
		})
	}
}

// Benchmark formatted logging methods
func BenchmarkFormattedLogging(b *testing.B) {
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    discardWriter{},
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
		Async:     false,
	})
	defer h.Close()

	log := logger.NewBuilder().
		WithHandler(h).
		WithLevel(core.InfoLevel).
		Build()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.Infof("test message %d %s", i, "value")
	}
}

// Benchmark context fields (using With())
func BenchmarkContextFields(b *testing.B) {
	tests := []struct {
		name       string
		fieldCount int
	}{
		{"NoContext", 0},
		{"1ContextField", 1},
		{"5ContextFields", 5},
		{"10ContextFields", 10},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
				Writer:    discardWriter{},
				Formatter: formatter.NewTextFormatter(formatter.Config{}),
				Async:     false,
			})
			defer h.Close()

			log := logger.NewBuilder().
				WithHandler(h).
				WithLevel(core.InfoLevel).
				Build()

			// Add context fields
			fields := make([]core.Field, tt.fieldCount)
			for i := 0; i < tt.fieldCount; i++ {
				fields[i] = logger.String("context_key", "context_value")
			}
			if tt.fieldCount > 0 {
				log = log.With(fields...)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				log.Info("test message", logger.String("key", "value"))
			}
		})
	}
}

// Benchmark entry pool recycling
func BenchmarkEntryPool(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		entry := core.GetEntry()
		entry.Level = core.InfoLevel
		entry.Message = "test"
		entry.Fields = append(entry.Fields, logger.String("key", "value"))
		core.PutEntry(entry)
	}
}

// Benchmark different log levels
func BenchmarkLogLevels(b *testing.B) {
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    discardWriter{},
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
		Async:     false,
	})
	defer h.Close()

	log := logger.NewBuilder().
		WithHandler(h).
		WithLevel(core.DebugLevel). // Enable all levels
		Build()

	tests := []struct {
		name string
		fn   func(string, ...core.Field)
	}{
		{"Debug", log.Debug},
		{"Info", log.Info},
		{"Warn", log.Warn},
		{"Error", log.Error},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				tt.fn("test message", logger.String("key", "value"))
			}
		})
	}
}

// Benchmark concurrent logging
func BenchmarkConcurrentLogging(b *testing.B) {
	tests := []struct {
		name       string
		goroutines int
	}{
		{"1Goroutine", 1},
		{"2Goroutines", 2},
		{"4Goroutines", 4},
		{"8Goroutines", 8},
		{"16Goroutines", 16},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
				Writer:     discardWriter{},
				Formatter:  formatter.NewTextFormatter(formatter.Config{}),
				Async:      true,
				BufferSize: 10000,
			})
			defer h.Close()

			log := logger.NewBuilder().
				WithHandler(h).
				WithLevel(core.InfoLevel).
				Build()

			b.ResetTimer()
			b.ReportAllocs()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					log.Info("test message",
						logger.String("key1", "value1"),
						logger.Int("key2", 42),
					)
				}
			})
		})
	}
}

// Benchmark file handler (writing to actual file)
func BenchmarkFileHandler(b *testing.B) {
	tmpFile, err := os.CreateTemp("", "nlog_benchmark_*.log")
	if err != nil {
		b.Fatal(err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	h, err := filehandler.NewFileHandler(filehandler.FileConfig{
		Filename:   tmpFile.Name(),
		Formatter:  formatter.NewTextFormatter(formatter.Config{}),
		Async:      true,
		BufferSize: 10000,
	})
	if err != nil {
		b.Fatal(err)
	}
	defer h.Close()

	log := logger.NewBuilder().
		WithHandler(h).
		WithLevel(core.InfoLevel).
		Build()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.Info("test message",
			logger.String("key1", "value1"),
			logger.Int("key2", i),
		)
	}
}

// Benchmark multi handler
func BenchmarkMultiHandler(b *testing.B) {
	h1 := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    discardWriter{},
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
		Async:     false,
	})
	defer h1.Close()

	h2 := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    discardWriter{},
		Formatter: formatter.NewJSONFormatter(formatter.Config{}),
		Async:     false,
	})
	defer h2.Close()

	multiH := multihandler.NewMultiHandler(h1, h2)
	defer multiH.Close()

	log := logger.NewBuilder().
		WithHandler(multiH).
		WithLevel(core.InfoLevel).
		Build()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.Info("test message",
			logger.String("key1", "value1"),
			logger.Int("key2", 42),
		)
	}
}

// Benchmark buffer pool efficiency
func BenchmarkBufferPool(b *testing.B) {
	msg := "test message"
	kvs := []byte(" key=value\n")

	b.Run("WithBuffer", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			buf.Grow(len(msg) + len(kvs))
			buf.WriteString(msg)
			buf.Write(kvs)

			out := buf.Bytes()

			sinkBytes = out
			atomic.AddUint64(&sinkU64, uint64(len(out)))

			runtime.KeepAlive(out)
		}
	})

	b.Run("WithoutBuffer_RawBytes", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			data := []byte("test message key=value\n")

			sinkBytes = data
			atomic.AddUint64(&sinkU64, uint64(len(data)))
			runtime.KeepAlive(data)
		}
	})
}

// Benchmark realistic application scenario
func BenchmarkRealisticScenario(b *testing.B) {
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:     discardWriter{},
		Formatter:  formatter.NewJSONFormatter(formatter.Config{}),
		Async:      true,
		BufferSize: 10000,
	})
	defer h.Close()

	// Simulate a web application logger with context
	baseLog := logger.NewBuilder().
		WithHandler(h).
		WithLevel(core.InfoLevel).
		WithFields(
			logger.String("service", "api-gateway"),
			logger.String("version", "1.0.0"),
			logger.String("env", "production"),
		).
		Build()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simulate request logging
		reqLog := baseLog.With(
			logger.String("request_id", "req-12345"),
			logger.String("method", "GET"),
			logger.String("path", "/api/users"),
		)

		reqLog.Info("request received",
			logger.Int("user_id", 42),
			logger.Duration("latency", time.Millisecond*150),
			logger.Int("status", 200),
		)
	}
}

// Benchmark error field creation
func BenchmarkErrorField(b *testing.B) {
	testErr := errors.New("test error")

	b.Run("WithError", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			f := logger.Err(testErr)

			sinkField = f

			atomic.AddUint64(&sinkU64, 1)
			runtime.KeepAlive(f)
		}
	})

	b.Run("WithNilError", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			f := logger.Err(nil)
			sinkField = f
			atomic.AddUint64(&sinkU64, 1)
			runtime.KeepAlive(f)
		}
	})
}

// Benchmark large message handling
func BenchmarkLargeMessages(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"Small_50B", 50},
		{"Medium_500B", 500},
		{"Large_5KB", 5000},
		{"VeryLarge_50KB", 50000},
	}

	for _, sz := range sizes {
		b.Run(sz.name, func(b *testing.B) {
			h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
				Writer:     discardWriter{},
				Formatter:  formatter.NewTextFormatter(formatter.Config{}),
				Async:      false,
				BufferSize: 10000,
			})
			defer h.Close()

			log := logger.NewBuilder().
				WithHandler(h).
				WithLevel(core.InfoLevel).
				Build()

			message := string(make([]byte, sz.size))

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				log.Info(message)
			}
		})
	}
}

// Benchmark WriterFormatter optimization
func BenchmarkWriterFormatter(b *testing.B) {
	entry := &core.Entry{
		Time:    time.Now(),
		Level:   core.InfoLevel,
		Message: "test message",
		Fields: []core.Field{
			logger.String("key1", "value1"),
			logger.Int("key2", 42),
			logger.Float64("key3", 3.14),
		},
	}

	b.Run("Format", func(b *testing.B) {
		f := formatter.NewTextFormatter(formatter.Config{})
		w := discardWriter{}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			data, _ := f.Format(entry)
			w.Write(data)
		}
	})

	b.Run("FormatTo", func(b *testing.B) {
		f := formatter.NewTextFormatter(formatter.Config{})
		w := discardWriter{}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			f.FormatTo(entry, w)
		}
	})
}

// Benchmark overflow policies
func BenchmarkOverflowPolicies(b *testing.B) {
	tests := []struct {
		name   string
		policy handler.OverflowPolicy
	}{
		{"DropNewest", handler.DropNewest},
		{"Block", handler.Block},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			policies := make(map[core.Level]handler.OverflowPolicy)
			policies[core.InfoLevel] = tt.policy

			h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
				Writer:         discardWriter{},
				Formatter:      formatter.NewTextFormatter(formatter.Config{}),
				Async:          true,
				BufferSize:     1, // Small buffer to test overflow
				OverflowPolicy: policies,
			})
			defer h.Close()

			log := logger.NewBuilder().
				WithHandler(h).
				WithLevel(core.InfoLevel).
				Build()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				log.Info("test message", logger.Int("i", i))
			}
		})
	}
}

// Benchmark different buffer sizes for async handlers
func BenchmarkBufferSizes(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("BufferSize%d", size), func(b *testing.B) {
			h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
				Writer:     discardWriter{},
				Formatter:  formatter.NewTextFormatter(formatter.Config{}),
				Async:      true,
				BufferSize: size,
			})
			defer h.Close()

			log := logger.NewBuilder().
				WithHandler(h).
				WithLevel(core.InfoLevel).
				Build()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				log.Info("test message", logger.Int("i", i))
			}
		})
	}
}

// Benchmark batch logging (multiple logs in sequence)
func BenchmarkBatchLogging(b *testing.B) {
	batchSizes := []int{1, 10, 100, 1000}

	for _, batchSize := range batchSizes {
		b.Run(fmt.Sprintf("Batch%d", batchSize), func(b *testing.B) {
			h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
				Writer:     discardWriter{},
				Formatter:  formatter.NewTextFormatter(formatter.Config{}),
				Async:      true,
				BufferSize: 10000,
			})
			defer h.Close()

			log := logger.NewBuilder().
				WithHandler(h).
				WithLevel(core.InfoLevel).
				Build()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				for j := 0; j < batchSize; j++ {
					log.Info("test message", logger.Int("batch", i), logger.Int("item", j))
				}
			}
		})
	}
}

// Benchmark multi-handler with different numbers of handlers
func BenchmarkMultiHandlerCount(b *testing.B) {
	counts := []int{2, 3, 5, 10}

	for _, count := range counts {
		b.Run(fmt.Sprintf("%dHandlers", count), func(b *testing.B) {
			handlers := make([]handler.Handler, count)
			for i := 0; i < count; i++ {
				handlers[i] = consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
					Writer:    discardWriter{},
					Formatter: formatter.NewTextFormatter(formatter.Config{}),
					Async:     false,
				})
				defer handlers[i].Close()
			}

			multiH := multihandler.NewMultiHandler(handlers...)
			defer multiH.Close()

			log := logger.NewBuilder().
				WithHandler(multiH).
				WithLevel(core.InfoLevel).
				Build()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				log.Info("test message", logger.Int("i", i))
			}
		})
	}
}

// Benchmark deeply nested context loggers
func BenchmarkNestedContextLoggers(b *testing.B) {
	depths := []int{1, 5, 10, 20}

	for _, depth := range depths {
		b.Run(fmt.Sprintf("Depth%d", depth), func(b *testing.B) {
			h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
				Writer:    discardWriter{},
				Formatter: formatter.NewTextFormatter(formatter.Config{}),
				Async:     false,
			})
			defer h.Close()

			log := logger.NewBuilder().
				WithHandler(h).
				WithLevel(core.InfoLevel).
				Build()

			// Create nested context
			for i := 0; i < depth; i++ {
				log = log.With(logger.String(fmt.Sprintf("context%d", i), "value"))
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				log.Info("test message")
			}
		})
	}
}

// Benchmark mixed field types (realistic scenario)
func BenchmarkMixedFieldTypes(b *testing.B) {
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    discardWriter{},
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
		Async:     false,
	})
	defer h.Close()

	log := logger.NewBuilder().
		WithHandler(h).
		WithLevel(core.InfoLevel).
		Build()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.Info("mixed fields",
			logger.String("user_id", "user123"),
			logger.Int("request_count", 42),
			logger.Float64("response_time", 123.45),
			logger.Bool("success", true),
			logger.Duration("latency", time.Millisecond*150),
			logger.Time("timestamp", time.Now()),
		)
	}
}

// Benchmark JSON formatter with different field counts
func BenchmarkJSONFormatterFields(b *testing.B) {
	fieldCounts := []int{0, 1, 5, 10, 20}

	for _, count := range fieldCounts {
		b.Run(fmt.Sprintf("%dFields", count), func(b *testing.B) {
			h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
				Writer:    discardWriter{},
				Formatter: formatter.NewJSONFormatter(formatter.Config{}),
				Async:     false,
			})
			defer h.Close()

			log := logger.NewBuilder().
				WithHandler(h).
				WithLevel(core.InfoLevel).
				Build()

			// Pre-create fields
			fields := make([]core.Field, count)
			for i := 0; i < count; i++ {
				fields[i] = logger.String(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i))
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				log.Info("test message", fields...)
			}
		})
	}
}

// Benchmark string concatenation in messages
func BenchmarkMessageConstruction(b *testing.B) {
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    discardWriter{},
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
		Async:     false,
	})
	defer h.Close()

	log := logger.NewBuilder().
		WithHandler(h).
		WithLevel(core.InfoLevel).
		Build()

	b.Run("StaticMessage", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			log.Info("static message")
		}
	})

	b.Run("FormattedMessage", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			log.Infof("formatted message %d", i)
		}
	})

	b.Run("MessageWithFields", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			log.Info("message", logger.Int("index", i))
		}
	})
}

// Benchmark all log levels in sequence (realistic usage)
func BenchmarkAllLevelsSequence(b *testing.B) {
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:     discardWriter{},
		Formatter:  formatter.NewTextFormatter(formatter.Config{}),
		Async:      true,
		BufferSize: 10000,
	})
	defer h.Close()

	log := logger.NewBuilder().
		WithHandler(h).
		WithLevel(core.DebugLevel).
		Build()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.Debug("debug message")
		log.Info("info message")
		log.Warn("warn message")
		log.Error("error message")
	}
}

func BenchmarkNlog_Parallel_NoFields_Text(b *testing.B) {
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    io.Discard,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
		Async:     false,
	})
	defer h.Close()

	log := logger.NewBuilder().
		WithHandler(h).
		WithLevel(core.InfoLevel).
		Build()
	defer log.Close()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			log.Info("parallel log")
		}
	})
}

func BenchmarkNlog_Parallel_NoFields_JSON(b *testing.B) {
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    io.Discard,
		Formatter: formatter.NewJSONFormatter(formatter.Config{}),
		Async:     false,
	})
	defer h.Close()

	log := logger.NewBuilder().
		WithHandler(h).
		WithLevel(core.InfoLevel).
		Build()
	defer log.Close()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			log.Info("parallel log")
		}
	})
}

func BenchmarkNlog_Parallel_NoFormatting_NoopHandler(b *testing.B) {
	h := newNoopHandler() // sync noop; just PutEntry back
	defer h.Close()

	log := logger.NewBuilder().
		WithHandler(h).
		WithLevel(core.InfoLevel).
		Build()
	defer log.Close()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			log.Info("parallel log")
		}
	})
}

func BenchmarkNlog_Parallel_WithFields_Text(b *testing.B) {
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    io.Discard,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
		Async:     false,
	})
	defer h.Close()

	log := logger.NewBuilder().
		WithHandler(h).
		WithLevel(core.InfoLevel).
		Build()
	defer log.Close()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			log.Info("parallel log",
				logger.String("key", "value"),
				logger.Int("count", 42),
			)
		}
	})
}

func BenchmarkNlog_Parallel_WithFields_JSON(b *testing.B) {
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    io.Discard,
		Formatter: formatter.NewJSONFormatter(formatter.Config{}),
		Async:     false,
	})
	defer h.Close()

	log := logger.NewBuilder().
		WithHandler(h).
		WithLevel(core.InfoLevel).
		Build()
	defer log.Close()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			log.Info("parallel log",
				logger.String("key", "value"),
				logger.Int("count", 42),
			)
		}
	})
}

// Benchmark coarse clock vs standard clock
func BenchmarkCoarseClock_InfoNoFields(b *testing.B) {
	tests := []struct {
		name        string
		coarseClock bool
	}{
		{"Standard", false},
		{"CoarseClock", true},
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
				Writer:    discardWriter{},
				Formatter: formatter.NewTextFormatter(formatter.Config{}),
				Async:     false,
			})
			defer h.Close()

			log := logger.NewBuilder().
				WithHandler(h).
				WithLevel(core.InfoLevel).
				WithCoarseClock(tt.coarseClock).
				Build()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				log.Info("test message")
			}
		})
	}
}

func BenchmarkCoarseClock_Info5Fields(b *testing.B) {
	tests := []struct {
		name        string
		coarseClock bool
	}{
		{"Standard", false},
		{"CoarseClock", true},
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
				Writer:    discardWriter{},
				Formatter: formatter.NewTextFormatter(formatter.Config{}),
				Async:     false,
			})
			defer h.Close()

			log := logger.NewBuilder().
				WithHandler(h).
				WithLevel(core.InfoLevel).
				WithCoarseClock(tt.coarseClock).
				Build()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				log.Info("test message",
					logger.String("key1", "value1"),
					logger.Int("key2", 42),
					logger.Float64("key3", 3.14),
					logger.Bool("key4", true),
					logger.String("key5", "value5"),
				)
			}
		})
	}
}

func BenchmarkCoarseClock_SyncVsAsync(b *testing.B) {
	tests := []struct {
		name        string
		async       bool
		coarseClock bool
	}{
		{"Sync_Standard", false, false},
		{"Sync_CoarseClock", false, true},
		{"Async_Standard", true, false},
		{"Async_CoarseClock", true, true},
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
				Writer:     discardWriter{},
				Formatter:  formatter.NewTextFormatter(formatter.Config{}),
				Async:      tt.async,
				BufferSize: 10000,
			})
			defer h.Close()

			log := logger.NewBuilder().
				WithHandler(h).
				WithLevel(core.InfoLevel).
				WithCoarseClock(tt.coarseClock).
				Build()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				log.Info("test message",
					logger.String("key1", "value1"),
					logger.Int("key2", i),
				)
			}
		})
	}
}
