package core

import (
	"testing"
)

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{DebugLevel, "DEBUG"},
		{InfoLevel, "INFO"},
		{WarnLevel, "WARN"},
		{ErrorLevel, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.String(); got != tt.want {
				t.Errorf("Level.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntryPool(t *testing.T) {
	// Get an entry from the pool
	e1 := GetEntry()
	if e1 == nil {
		t.Fatal("GetEntry() returned nil")
	}

	// Verify initial state
	if len(e1.Fields) != 0 {
		t.Errorf("Expected empty fields, got %d", len(e1.Fields))
	}

	// Add some data
	e1.Message = "test"
	e1.Fields = append(e1.Fields, Field{Key: "test", Str: "value"})

	// Return to pool
	PutEntry(e1)

	// Get another entry
	e2 := GetEntry()
	if e2 == nil {
		t.Fatal("GetEntry() returned nil after PutEntry()")
	}

	// Verify it's clean
	if e2.Message != "" {
		t.Errorf("Expected empty message after pool reset, got %q", e2.Message)
	}
	if len(e2.Fields) != 0 {
		t.Errorf("Expected empty fields after pool reset, got %d", len(e2.Fields))
	}
}

func TestGetCaller(t *testing.T) {
	caller := GetCaller(0)
	if !caller.Defined {
		t.Fatal("GetCaller() returned undefined CallerInfo")
	}

	if caller.File == "" {
		t.Error("Expected non-empty file")
	}
	if caller.ShortFile == "" {
		t.Error("Expected non-empty short file")
	}
	if caller.Line == 0 {
		t.Error("Expected non-zero line number")
	}
	if caller.Function == "" {
		t.Error("Expected non-empty function name")
	}
}

func BenchmarkGetEntry(b *testing.B) {
	for i := 0; i < b.N; i++ {
		e := GetEntry()
		PutEntry(e)
	}
}

func BenchmarkGetEntryWithFields(b *testing.B) {
	for i := 0; i < b.N; i++ {
		e := GetEntry()
		e.Message = "test message"
		e.Level = InfoLevel
		e.Fields = append(e.Fields, Field{Key: "key1", Str: "value1"})
		e.Fields = append(e.Fields, Field{Key: "key2", Int64: 42})
		PutEntry(e)
	}
}
