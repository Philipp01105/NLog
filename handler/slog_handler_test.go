package handler

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/Philipp01105/NLog/core"
	"github.com/Philipp01105/NLog/formatter"
)

func TestSlogHandler_Enabled(t *testing.T) {
	h := NewConsoleHandler(ConsoleConfig{
		Writer:    &bytes.Buffer{},
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})

	sh := NewSlogHandler(h, core.InfoLevel)

	if sh.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("Debug should not be enabled when level is Info")
	}
	if !sh.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("Info should be enabled when level is Info")
	}
	if !sh.Enabled(context.Background(), slog.LevelWarn) {
		t.Error("Warn should be enabled when level is Info")
	}
	if !sh.Enabled(context.Background(), slog.LevelError) {
		t.Error("Error should be enabled when level is Info")
	}
}

func TestSlogHandler_Handle(t *testing.T) {
	var buf bytes.Buffer
	h := NewConsoleHandler(ConsoleConfig{
		Writer:    &buf,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})

	sh := NewSlogHandler(h, core.DebugLevel)
	logger := slog.New(sh)

	logger.Info("test message", "key", "value", "count", 42)

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected 'test message' in output, got: %s", output)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("Expected 'key=value' in output, got: %s", output)
	}
	if !strings.Contains(output, "count=42") {
		t.Errorf("Expected 'count=42' in output, got: %s", output)
	}
}

func TestSlogHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	h := NewConsoleHandler(ConsoleConfig{
		Writer:    &buf,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})

	sh := NewSlogHandler(h, core.DebugLevel)
	logger := slog.New(sh).With("request_id", "req-123")

	logger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "request_id=req-123") {
		t.Errorf("Expected 'request_id=req-123' in output, got: %s", output)
	}
}

func TestSlogHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	h := NewConsoleHandler(ConsoleConfig{
		Writer:    &buf,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})

	sh := NewSlogHandler(h, core.DebugLevel)
	logger := slog.New(sh).WithGroup("auth")

	logger.Info("test message", "user_id", 123)

	output := buf.String()
	if !strings.Contains(output, "auth.user_id=123") {
		t.Errorf("Expected 'auth.user_id=123' in output, got: %s", output)
	}
}

func TestSlogHandler_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	h := NewConsoleHandler(ConsoleConfig{
		Writer:    &buf,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})

	sh := NewSlogHandler(h, core.InfoLevel)
	logger := slog.New(sh)

	logger.Debug("should not appear")
	if buf.Len() > 0 {
		t.Error("Debug message should not have been logged")
	}

	logger.Info("should appear")
	if !strings.Contains(buf.String(), "should appear") {
		t.Errorf("Expected 'should appear' in output, got: %s", buf.String())
	}
}

func TestSlogLevelToCore(t *testing.T) {
	tests := []struct {
		slogLevel slog.Level
		coreLevel core.Level
	}{
		{slog.LevelDebug, core.DebugLevel},
		{slog.LevelInfo, core.InfoLevel},
		{slog.LevelWarn, core.WarnLevel},
		{slog.LevelError, core.ErrorLevel},
	}

	for _, tt := range tests {
		got := slogLevelToCore(tt.slogLevel)
		if got != tt.coreLevel {
			t.Errorf("slogLevelToCore(%v) = %v, want %v", tt.slogLevel, got, tt.coreLevel)
		}
	}
}
