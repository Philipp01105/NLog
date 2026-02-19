package formatter

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/philipp01105/nlog/core"
)

func TestTextFormatter_Basic(t *testing.T) {
	f := NewTextFormatter(Config{})

	entry := &core.Entry{
		Time:    time.Date(2026, 2, 18, 13, 0, 0, 0, time.UTC),
		Level:   core.InfoLevel,
		Message: "test message",
	}

	result, err := f.Format(entry)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)
	if !strings.Contains(output, "[INFO]") {
		t.Errorf("Expected '[INFO]' in output, got: %s", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected 'test message' in output, got: %s", output)
	}
}

func TestTextFormatter_WithFields(t *testing.T) {
	f := NewTextFormatter(Config{})

	entry := &core.Entry{
		Time:    time.Now(),
		Level:   core.InfoLevel,
		Message: "test",
		Fields: []core.Field{
			{Key: "key1", Type: core.StringType, Str: "value1"},
			{Key: "key2", Type: core.IntType, Int64: 42},
		},
	}

	result, err := f.Format(entry)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)
	if !strings.Contains(output, "key1=value1") {
		t.Errorf("Expected 'key1=value1' in output, got: %s", output)
	}
	if !strings.Contains(output, "key2=42") {
		t.Errorf("Expected 'key2=42' in output, got: %s", output)
	}
}

func TestTextFormatter_WithCaller(t *testing.T) {
	f := NewTextFormatter(Config{IncludeCaller: true})

	entry := &core.Entry{
		Time:    time.Now(),
		Level:   core.InfoLevel,
		Message: "test",
		Caller: core.CallerInfo{
			File:      "/path/to/file.go",
			ShortFile: "file.go",
			Line:      123,
			Function:  "main.main",
			Defined:   true,
		},
	}

	result, err := f.Format(entry)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)
	if !strings.Contains(output, "file.go:123") {
		t.Errorf("Expected caller info in output, got: %s", output)
	}
}

func TestJSONFormatter_Basic(t *testing.T) {
	f := NewJSONFormatter(Config{})

	entry := &core.Entry{
		Time:    time.Date(2026, 2, 18, 13, 0, 0, 0, time.UTC),
		Level:   core.InfoLevel,
		Message: "test message",
	}

	result, err := f.Format(entry)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Verify it's valid JSON
	var data map[string]interface{}
	if err := json.Unmarshal(result, &data); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if data["level"] != "INFO" {
		t.Errorf("Expected level 'INFO', got: %v", data["level"])
	}
	if data["message"] != "test message" {
		t.Errorf("Expected message 'test message', got: %v", data["message"])
	}
}

func TestJSONFormatter_WithFields(t *testing.T) {
	f := NewJSONFormatter(Config{})

	entry := &core.Entry{
		Time:    time.Now(),
		Level:   core.InfoLevel,
		Message: "test",
		Fields: []core.Field{
			{Key: "str", Type: core.StringType, Str: "value"},
			{Key: "int", Type: core.IntType, Int64: 42},
			{Key: "bool", Type: core.BoolType, Int64: 1},
		},
	}

	result, err := f.Format(entry)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result, &data); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if data["str"] != "value" {
		t.Errorf("Expected str='value', got: %v", data["str"])
	}
	if data["int"] != float64(42) { // JSON numbers are float64
		t.Errorf("Expected int=42, got: %v", data["int"])
	}
	if data["bool"] != true {
		t.Errorf("Expected bool=true, got: %v", data["bool"])
	}
}

func TestJSONFormatter_WithCaller(t *testing.T) {
	f := NewJSONFormatter(Config{IncludeCaller: true})

	entry := &core.Entry{
		Time:    time.Now(),
		Level:   core.InfoLevel,
		Message: "test",
		Caller: core.CallerInfo{
			File:      "/path/to/file.go",
			ShortFile: "file.go",
			Line:      123,
			Function:  "main.main",
			Defined:   true,
		},
	}

	result, err := f.Format(entry)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result, &data); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	caller, ok := data["caller"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected caller object in JSON")
	}

	if caller["file"] != "file.go" {
		t.Errorf("Expected file='file.go', got: %v", caller["file"])
	}
	if caller["line"] != float64(123) {
		t.Errorf("Expected line=123, got: %v", caller["line"])
	}
}

func BenchmarkTextFormatter(b *testing.B) {
	f := NewTextFormatter(Config{})
	entry := &core.Entry{
		Time:    time.Now(),
		Level:   core.InfoLevel,
		Message: "test message",
		Fields: []core.Field{
			{Key: "key1", Type: core.StringType, Str: "value1"},
			{Key: "key2", Type: core.IntType, Int64: 42},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.Format(entry)
	}
}

func BenchmarkJSONFormatter(b *testing.B) {
	f := NewJSONFormatter(Config{})
	entry := &core.Entry{
		Time:    time.Now(),
		Level:   core.InfoLevel,
		Message: "test message",
		Fields: []core.Field{
			{Key: "key1", Type: core.StringType, Str: "value1"},
			{Key: "key2", Type: core.IntType, Int64: 42},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.Format(entry)
	}
}
