package benchmark

import (
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/philipp01105/nlog/core"
	"github.com/philipp01105/nlog/formatter"
	"github.com/philipp01105/nlog/handler"
	"github.com/philipp01105/nlog/logger"
)

// ---------------------------------------------------------------------------
// Helpers – identical sink for every framework (io.Discard / no-op writer)
// ---------------------------------------------------------------------------

// newNlogLogger returns an nlog logger that writes JSON to io.Discard.
func newNlogLogger() *logger.Logger {
	h := handler.NewConsoleHandler(handler.ConsoleConfig{
		Writer:    io.Discard,
		Formatter: formatter.NewJSONFormatter(formatter.Config{}),
		Async:     false,
	})
	return logger.NewBuilder().
		WithHandler(h).
		WithLevel(core.DebugLevel).
		Build()
}

// newZapLogger returns a zap.Logger that writes JSON to io.Discard.
func newZapLogger() *zap.Logger {
	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := zapcore.NewCore(enc, zapcore.AddSync(io.Discard), zap.DebugLevel)
	return zap.New(core)
}

// newSlogLogger returns an slog.Logger that writes JSON to io.Discard.
func newSlogLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// newLogrusLogger returns a logrus.Logger that writes JSON to io.Discard.
func newLogrusLogger() *logrus.Logger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	l.SetFormatter(&logrus.JSONFormatter{})
	l.SetLevel(logrus.DebugLevel)
	return l
}

// newZerologLogger returns a zerolog.Logger that writes JSON to io.Discard.
func newZerologLogger() zerolog.Logger {
	return zerolog.New(io.Discard).With().Timestamp().Logger().Level(zerolog.DebugLevel)
}

// ---------------------------------------------------------------------------
// Scenario 1 – Info message, no fields
// ---------------------------------------------------------------------------

func BenchmarkCompetitive_InfoNoFields(b *testing.B) {
	b.Run("nlog", func(b *testing.B) {
		l := newNlogLogger()
		defer l.Close()
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.Info("info message")
		}
	})

	b.Run("zap", func(b *testing.B) {
		l := newZapLogger()
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.Info("info message")
		}
	})

	b.Run("slog", func(b *testing.B) {
		l := newSlogLogger()
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.Info("info message")
		}
	})

	b.Run("logrus", func(b *testing.B) {
		l := newLogrusLogger()
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.Info("info message")
		}
	})

	b.Run("zerolog", func(b *testing.B) {
		l := newZerologLogger()
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.Info().Msg("info message")
		}
	})
}

// ---------------------------------------------------------------------------
// Scenario 2 – Structured logging with common fields
// ---------------------------------------------------------------------------

func BenchmarkCompetitive_InfoWithFields(b *testing.B) {
	b.Run("nlog", func(b *testing.B) {
		l := newNlogLogger()
		defer l.Close()
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.Info("request handled",
				logger.String("method", "GET"),
				logger.String("path", "/api/users"),
				logger.Int("status", 200),
				logger.Duration("latency", 150*time.Millisecond),
			)
		}
	})

	b.Run("zap", func(b *testing.B) {
		l := newZapLogger()
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.Info("request handled",
				zap.String("method", "GET"),
				zap.String("path", "/api/users"),
				zap.Int("status", 200),
				zap.Duration("latency", 150*time.Millisecond),
			)
		}
	})

	b.Run("slog", func(b *testing.B) {
		l := newSlogLogger()
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.Info("request handled",
				slog.String("method", "GET"),
				slog.String("path", "/api/users"),
				slog.Int("status", 200),
				slog.Duration("latency", 150*time.Millisecond),
			)
		}
	})

	b.Run("logrus", func(b *testing.B) {
		l := newLogrusLogger()
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.WithFields(logrus.Fields{
				"method":  "GET",
				"path":    "/api/users",
				"status":  200,
				"latency": 150 * time.Millisecond,
			}).Info("request handled")
		}
	})

	b.Run("zerolog", func(b *testing.B) {
		l := newZerologLogger()
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.Info().
				Str("method", "GET").
				Str("path", "/api/users").
				Int("status", 200).
				Dur("latency", 150*time.Millisecond).
				Msg("request handled")
		}
	})
}

// ---------------------------------------------------------------------------
// Scenario 3 – Disabled level (measure level-check overhead)
// ---------------------------------------------------------------------------

func BenchmarkCompetitive_DisabledLevel(b *testing.B) {
	b.Run("nlog", func(b *testing.B) {
		h := handler.NewConsoleHandler(handler.ConsoleConfig{
			Writer:    io.Discard,
			Formatter: formatter.NewJSONFormatter(formatter.Config{}),
			Async:     false,
		})
		l := logger.NewBuilder().
			WithHandler(h).
			WithLevel(core.ErrorLevel).
			Build()
		defer l.Close()
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.Debug("should be skipped", logger.String("key", "value"))
		}
	})

	b.Run("zap", func(b *testing.B) {
		enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
		core := zapcore.NewCore(enc, zapcore.AddSync(io.Discard), zap.ErrorLevel)
		l := zap.New(core)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.Debug("should be skipped", zap.String("key", "value"))
		}
	})

	b.Run("slog", func(b *testing.B) {
		l := slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.Debug("should be skipped", slog.String("key", "value"))
		}
	})

	b.Run("logrus", func(b *testing.B) {
		l := logrus.New()
		l.SetOutput(io.Discard)
		l.SetFormatter(&logrus.JSONFormatter{})
		l.SetLevel(logrus.ErrorLevel)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.WithField("key", "value").Debug("should be skipped")
		}
	})

	b.Run("zerolog", func(b *testing.B) {
		l := zerolog.New(io.Discard).Level(zerolog.ErrorLevel)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.Debug().Str("key", "value").Msg("should be skipped")
		}
	})
}

// ---------------------------------------------------------------------------
// Scenario 4 – Accumulated context fields (child logger / With)
// ---------------------------------------------------------------------------

func BenchmarkCompetitive_AccumulatedContext(b *testing.B) {
	b.Run("nlog", func(b *testing.B) {
		l := newNlogLogger()
		defer l.Close()
		cl := l.With(
			logger.String("service", "api"),
			logger.String("env", "prod"),
			logger.String("version", "1.0.0"),
		)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			cl.Info("request", logger.Int("status", 200))
		}
	})

	b.Run("zap", func(b *testing.B) {
		l := newZapLogger().With(
			zap.String("service", "api"),
			zap.String("env", "prod"),
			zap.String("version", "1.0.0"),
		)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.Info("request", zap.Int("status", 200))
		}
	})

	b.Run("slog", func(b *testing.B) {
		l := newSlogLogger().With(
			slog.String("service", "api"),
			slog.String("env", "prod"),
			slog.String("version", "1.0.0"),
		)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.Info("request", slog.Int("status", 200))
		}
	})

	b.Run("logrus", func(b *testing.B) {
		l := newLogrusLogger().WithFields(logrus.Fields{
			"service": "api",
			"env":     "prod",
			"version": "1.0.0",
		})
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.WithField("status", 200).Info("request")
		}
	})

	b.Run("zerolog", func(b *testing.B) {
		l := newZerologLogger().With().
			Str("service", "api").
			Str("env", "prod").
			Str("version", "1.0.0").
			Logger()
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.Info().Int("status", 200).Msg("request")
		}
	})
}

// ---------------------------------------------------------------------------
// Scenario 5 – Parallel / high-concurrency logging
// ---------------------------------------------------------------------------

func BenchmarkCompetitive_Parallel(b *testing.B) {
	b.Run("nlog", func(b *testing.B) {
		l := newNlogLogger()
		defer l.Close()
		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				l.Info("parallel log",
					logger.String("key", "value"),
					logger.Int("count", 42),
				)
			}
		})
	})

	b.Run("zap", func(b *testing.B) {
		l := newZapLogger()
		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				l.Info("parallel log",
					zap.String("key", "value"),
					zap.Int("count", 42),
				)
			}
		})
	})

	b.Run("slog", func(b *testing.B) {
		l := newSlogLogger()
		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				l.Info("parallel log",
					slog.String("key", "value"),
					slog.Int("count", 42),
				)
			}
		})
	})

	b.Run("logrus", func(b *testing.B) {
		l := newLogrusLogger()
		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				l.WithFields(logrus.Fields{
					"key":   "value",
					"count": 42,
				}).Info("parallel log")
			}
		})
	})

	b.Run("zerolog", func(b *testing.B) {
		l := newZerologLogger()
		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				l.Info().
					Str("key", "value").
					Int("count", 42).
					Msg("parallel log")
			}
		})
	})
}

// ---------------------------------------------------------------------------
// Scenario 6 – File output (real I/O, equal conditions)
// ---------------------------------------------------------------------------

func BenchmarkCompetitive_FileOutput(b *testing.B) {
	b.Run("nlog", func(b *testing.B) {
		f, err := os.CreateTemp(b.TempDir(), "bench-nlog-*.log")
		if err != nil {
			b.Fatal(err)
		}
		h := handler.NewConsoleHandler(handler.ConsoleConfig{
			Writer:    f,
			Formatter: formatter.NewJSONFormatter(formatter.Config{}),
			Async:     false,
		})
		l := logger.NewBuilder().
			WithHandler(h).
			WithLevel(core.InfoLevel).
			Build()
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.Info("file log", logger.String("key", "value"))
		}
		b.StopTimer()
		l.Close()
		f.Close()
	})

	b.Run("zap", func(b *testing.B) {
		f, err := os.CreateTemp(b.TempDir(), "bench-zap-*.log")
		if err != nil {
			b.Fatal(err)
		}
		enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
		core := zapcore.NewCore(enc, zapcore.AddSync(f), zap.InfoLevel)
		l := zap.New(core)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.Info("file log", zap.String("key", "value"))
		}
		b.StopTimer()
		l.Sync()
		f.Close()
	})

	b.Run("slog", func(b *testing.B) {
		f, err := os.CreateTemp(b.TempDir(), "bench-slog-*.log")
		if err != nil {
			b.Fatal(err)
		}
		l := slog.New(slog.NewJSONHandler(f, &slog.HandlerOptions{Level: slog.LevelInfo}))
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.Info("file log", slog.String("key", "value"))
		}
		b.StopTimer()
		f.Close()
	})

	b.Run("logrus", func(b *testing.B) {
		f, err := os.CreateTemp(b.TempDir(), "bench-logrus-*.log")
		if err != nil {
			b.Fatal(err)
		}
		l := logrus.New()
		l.SetOutput(f)
		l.SetFormatter(&logrus.JSONFormatter{})
		l.SetLevel(logrus.InfoLevel)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.WithField("key", "value").Info("file log")
		}
		b.StopTimer()
		f.Close()
	})

	b.Run("zerolog", func(b *testing.B) {
		f, err := os.CreateTemp(b.TempDir(), "bench-zerolog-*.log")
		if err != nil {
			b.Fatal(err)
		}
		l := zerolog.New(f).With().Timestamp().Logger().Level(zerolog.InfoLevel)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.Info().Str("key", "value").Msg("file log")
		}
		b.StopTimer()
		f.Close()
	})
}
