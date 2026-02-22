package logger

import (
	"bytes"
	"strings"
	"testing"

	"github.com/philipp01105/nlog/formatter"
	"github.com/philipp01105/nlog/handler/consolehandler"
)

func TestLogger_LevelGate(t *testing.T) {
	var buf bytes.Buffer
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    &buf,
		Async:     false, // Synchronous for testing
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})

	logger := NewBuilder().
		WithHandler(h).
		WithLevel(InfoLevel).
		Build()

	// Debug should not be logged (below Info level)
	logger.Debug("debug message")
	if buf.Len() > 0 {
		t.Error("Debug message was logged when level is Info")
	}

	// Info should be logged
	logger.Info("info message")
	if buf.Len() == 0 {
		t.Error("Info message was not logged")
	}
	if !strings.Contains(buf.String(), "info message") {
		t.Errorf("Expected 'info message' in output, got: %s", buf.String())
	}

	buf.Reset()

	// Warn should be logged
	logger.Warn("warn message")
	if !strings.Contains(buf.String(), "warn message") {
		t.Errorf("Expected 'warn message' in output, got: %s", buf.String())
	}

	buf.Reset()

	// Error should be logged
	logger.Error("error message")
	if !strings.Contains(buf.String(), "error message") {
		t.Errorf("Expected 'error message' in output, got: %s", buf.String())
	}
}

func TestLogger_With(t *testing.T) {
	var buf bytes.Buffer
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    &buf,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})

	logger := NewBuilder().
		WithHandler(h).
		WithLevel(InfoLevel).
		WithFields(String("app", "test")).
		Build()

	// Create child logger with additional fields
	childLogger := logger.With(String("request_id", "123"))

	childLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "app=test") {
		t.Errorf("Expected 'app=test' in output, got: %s", output)
	}
	if !strings.Contains(output, "request_id=123") {
		t.Errorf("Expected 'request_id=123' in output, got: %s", output)
	}
}

func TestLogger_Fields(t *testing.T) {
	var buf bytes.Buffer
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    &buf,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})

	logger := NewBuilder().
		WithHandler(h).
		WithLevel(InfoLevel).
		Build()

	logger.Info("test",
		String("str", "value"),
		Int("int", 42),
		Bool("bool", true),
		Float64("float", 3.14),
	)

	output := buf.String()
	if !strings.Contains(output, "str=value") {
		t.Errorf("Expected 'str=value' in output, got: %s", output)
	}
	if !strings.Contains(output, "int=42") {
		t.Errorf("Expected 'int=42' in output, got: %s", output)
	}
	if !strings.Contains(output, "bool=true") {
		t.Errorf("Expected 'bool=true' in output, got: %s", output)
	}
	if !strings.Contains(output, "float=3.14") {
		t.Errorf("Expected 'float=3.14' in output, got: %s", output)
	}
}

func TestLogger_FormattedLogging(t *testing.T) {
	var buf bytes.Buffer
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    &buf,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})

	logger := NewBuilder().
		WithHandler(h).
		WithLevel(InfoLevel).
		Build()

	logger.Infof("User %s logged in with ID %d", "alice", 123)

	output := buf.String()
	if !strings.Contains(output, "User alice logged in with ID 123") {
		t.Errorf("Expected formatted message in output, got: %s", output)
	}
}

func TestLogger_ImmutableWith(t *testing.T) {
	var buf bytes.Buffer
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    &buf,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})

	parent := NewBuilder().
		WithHandler(h).
		WithLevel(InfoLevel).
		WithFields(String("parent", "value")).
		Build()

	child := parent.With(String("child", "value"))

	// Parent should only have parent field
	parent.Info("parent message")
	parentOutput := buf.String()
	if !strings.Contains(parentOutput, "parent=value") {
		t.Error("Parent logger should have parent field")
	}
	if strings.Contains(parentOutput, "child=value") {
		t.Error("Parent logger should not have child field")
	}

	buf.Reset()

	// Child should have both fields
	child.Info("child message")
	childOutput := buf.String()
	if !strings.Contains(childOutput, "parent=value") {
		t.Error("Child logger should have parent field")
	}
	if !strings.Contains(childOutput, "child=value") {
		t.Error("Child logger should have child field")
	}
}

func BenchmarkLogger_LevelCheck(b *testing.B) {
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    &bytes.Buffer{},
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})

	logger := NewBuilder().
		WithHandler(h).
		WithLevel(InfoLevel).
		Build()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Should exit early due to level check
		logger.Debug("debug message", String("key", "value"))
	}
}

func BenchmarkLogger_Info(b *testing.B) {
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    &bytes.Buffer{},
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})

	logger := NewBuilder().
		WithHandler(h).
		WithLevel(InfoLevel).
		Build()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("test message", String("key", "value"))
	}
}

func BenchmarkLogger_InfoWithFields(b *testing.B) {
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    &bytes.Buffer{},
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})

	logger := NewBuilder().
		WithHandler(h).
		WithLevel(InfoLevel).
		Build()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("test message",
			String("str", "value"),
			Int("int", 42),
			Bool("bool", true),
			Float64("float", 3.14),
		)
	}
}

func TestLogger_Fatal(t *testing.T) {
	var buf bytes.Buffer
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    &buf,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})

	log := NewBuilder().
		WithHandler(h).
		WithLevel(DebugLevel).
		Build()

	// Override osExit to capture exit code instead of actually exiting
	exitCode := -1
	origExit := osExit
	osExit = func(code int) { exitCode = code }
	defer func() { osExit = origExit }()

	log.Fatal("fatal error", String("key", "value"))

	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(buf.String(), "fatal error") {
		t.Errorf("Expected 'fatal error' in output, got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "FATAL") {
		t.Errorf("Expected 'FATAL' in output, got: %s", buf.String())
	}
}

func TestLogger_Panic(t *testing.T) {
	var buf bytes.Buffer
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    &buf,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})

	log := NewBuilder().
		WithHandler(h).
		WithLevel(DebugLevel).
		Build()

	defer func() {
		r := recover()
		if r == nil {
			t.Error("Expected panic, got nil")
		}
		if r != "panic message" {
			t.Errorf("Expected panic with 'panic message', got: %v", r)
		}
		if !strings.Contains(buf.String(), "panic message") {
			t.Errorf("Expected 'panic message' in output, got: %s", buf.String())
		}
		if !strings.Contains(buf.String(), "PANIC") {
			t.Errorf("Expected 'PANIC' in output, got: %s", buf.String())
		}
	}()

	log.Panic("panic message")
}

func TestLogger_WithCoarseClock(t *testing.T) {
	var buf bytes.Buffer
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    &buf,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})

	log := NewBuilder().
		WithHandler(h).
		WithLevel(InfoLevel).
		WithCoarseClock(true).
		Build()

	log.Info("coarse clock message")
	output := buf.String()
	if !strings.Contains(output, "coarse clock message") {
		t.Errorf("Expected 'coarse clock message' in output, got: %s", output)
	}

	buf.Reset()

	// Also test with fields (non-fast path)
	log.Info("with field", String("key", "value"))
	output = buf.String()
	if !strings.Contains(output, "with field") {
		t.Errorf("Expected 'with field' in output, got: %s", output)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("Expected 'key=value' in output, got: %s", output)
	}
}

func TestLogger_CoarseClockWith(t *testing.T) {
	var buf bytes.Buffer
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    &buf,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})

	parent := NewBuilder().
		WithHandler(h).
		WithLevel(InfoLevel).
		WithCoarseClock(true).
		Build()

	child := parent.With(String("child", "value"))
	child.Info("child message")
	output := buf.String()
	if !strings.Contains(output, "child message") {
		t.Errorf("Expected 'child message' in output, got: %s", output)
	}
}

func TestParseLevel_FatalPanic(t *testing.T) {
	if ParseLevel("FATAL") != FatalLevel {
		t.Error("Expected FatalLevel for 'FATAL'")
	}
	if ParseLevel("PANIC") != PanicLevel {
		t.Error("Expected PanicLevel for 'PANIC'")
	}
}
