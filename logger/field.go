package logger

import (
	"time"

	"github.com/philipp01105/nlog/core"
)

// Field helper functions for convenience

// String creates a string field
func String(key, val string) core.Field {
	return core.Field{Key: key, Type: core.StringType, Str: val}
}

// Int creates an int field
func Int(key string, val int) core.Field {
	return core.Field{Key: key, Type: core.IntType, Int64: int64(val)}
}

// Int64 creates an int64 field
func Int64(key string, val int64) core.Field {
	return core.Field{Key: key, Type: core.Int64Type, Int64: val}
}

// Float64 creates a float64 field
func Float64(key string, val float64) core.Field {
	return core.Field{Key: key, Type: core.Float64Type, Float64: val}
}

// Bool creates a bool field
func Bool(key string, val bool) core.Field {
	int64Val := int64(0)
	if val {
		int64Val = 1
	}
	return core.Field{Key: key, Type: core.BoolType, Int64: int64Val}
}

// Time creates a time field
func Time(key string, val time.Time) core.Field {
	return core.Field{Key: key, Type: core.TimeType, Int64: val.UnixNano()}
}

// Duration creates a duration field
func Duration(key string, val time.Duration) core.Field {
	return core.Field{Key: key, Type: core.DurationType, Int64: int64(val)}
}

// Err creates an error field
func Err(err error) core.Field {
	if err == nil {
		return core.Field{Key: "error", Type: core.ErrorType, Str: ""}
	}
	return core.Field{Key: "error", Type: core.ErrorType, Str: err.Error()}
}

// Any creates a field with any value.
// For common primitive types, it uses typed fields to avoid boxing allocations.
func Any(key string, val interface{}) core.Field {
	switch v := val.(type) {
	case string:
		return core.Field{Key: key, Type: core.StringType, Str: v}
	case int:
		return core.Field{Key: key, Type: core.IntType, Int64: int64(v)}
	case int64:
		return core.Field{Key: key, Type: core.Int64Type, Int64: v}
	case float64:
		return core.Field{Key: key, Type: core.Float64Type, Float64: v}
	case bool:
		int64Val := int64(0)
		if v {
			int64Val = 1
		}
		return core.Field{Key: key, Type: core.BoolType, Int64: int64Val}
	case time.Duration:
		return core.Field{Key: key, Type: core.DurationType, Int64: int64(v)}
	case time.Time:
		return core.Field{Key: key, Type: core.TimeType, Int64: v.UnixNano()}
	case error:
		if v == nil {
			return core.Field{Key: key, Type: core.ErrorType, Str: ""}
		}
		return core.Field{Key: key, Type: core.ErrorType, Str: v.Error()}
	default:
		return core.Field{Key: key, Type: core.AnyType, Any: val}
	}
}
