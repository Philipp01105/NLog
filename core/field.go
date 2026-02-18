package core

import (
	"fmt"
	"strconv"
	"time"
)

// FieldType represents the type of a field value
type FieldType uint8

const (
	StringType FieldType = iota
	IntType
	Int64Type
	Float64Type
	BoolType
	TimeType
	DurationType
	ErrorType
	AnyType
)

// Field represents a key-value pair for structured logging
type Field struct {
	Key     string
	Type    FieldType
	Int64   int64
	Float64 float64
	Str     string
	Any     interface{}
}

// StringValue returns the string representation of a field's value
func (f Field) StringValue() string {
	switch f.Type {
	case StringType:
		return f.Str
	case IntType, Int64Type:
		return strconv.FormatInt(f.Int64, 10)
	case Float64Type:
		return strconv.FormatFloat(f.Float64, 'f', -1, 64)
	case BoolType:
		return strconv.FormatBool(f.Int64 == 1)
	case TimeType:
		return time.Unix(0, f.Int64).Format(time.RFC3339)
	case DurationType:
		return time.Duration(f.Int64).String()
	case ErrorType:
		return f.Str
	case AnyType:
		return fmt.Sprintf("%v", f.Any)
	default:
		return ""
	}
}
