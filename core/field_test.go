package core

import (
	"testing"
	"time"
)

func TestField_StringValue(t *testing.T) {
	tests := []struct {
		name  string
		field Field
		want  string
	}{
		{
			name:  "String field",
			field: Field{Type: StringType, Str: "hello"},
			want:  "hello",
		},
		{
			name:  "Int field",
			field: Field{Type: IntType, Int64: 42},
			want:  "42",
		},
		{
			name:  "Int64 field",
			field: Field{Type: Int64Type, Int64: 1234567890},
			want:  "1234567890",
		},
		{
			name:  "Bool field (true)",
			field: Field{Type: BoolType, Int64: 1},
			want:  "true",
		},
		{
			name:  "Bool field (false)",
			field: Field{Type: BoolType, Int64: 0},
			want:  "false",
		},
		{
			name:  "Float64 field",
			field: Field{Type: Float64Type, Float64: 3.14},
			want:  "3.14",
		},
		{
			name:  "Duration field",
			field: Field{Type: DurationType, Int64: int64(5 * time.Second)},
			want:  "5s",
		},
		{
			name:  "Error field",
			field: Field{Type: ErrorType, Str: "an error occurred"},
			want:  "an error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.field.StringValue(); got != tt.want {
				t.Errorf("Field.StringValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkFieldStringValue(b *testing.B) {
	fields := []Field{
		{Type: StringType, Str: "test"},
		{Type: IntType, Int64: 42},
		{Type: BoolType, Int64: 1},
		{Type: Float64Type, Float64: 3.14},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, f := range fields {
			_ = f.StringValue()
		}
	}
}
