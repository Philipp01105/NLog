package handler

import (
	"context"
	"log/slog"

	"github.com/Philipp01105/logging-framework/core"
)

// SlogHandler is an adapter that implements slog.Handler using a logging-framework Handler.
// This allows the logging framework to be used as a drop-in replacement for log/slog.
type SlogHandler struct {
	handler Handler
	level   core.Level
	attrs   []core.Field
	group   string
}

// NewSlogHandler creates a new slog.Handler adapter wrapping the given Handler.
func NewSlogHandler(h Handler, level core.Level) *SlogHandler {
	return &SlogHandler{
		handler: h,
		level:   level,
	}
}

// Enabled reports whether the handler handles records at the given level.
func (s *SlogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return slogLevelToCore(level) >= s.level
}

// Handle processes a slog.Record by converting it to a core.Entry and passing it to the wrapped handler.
func (s *SlogHandler) Handle(_ context.Context, record slog.Record) error {
	entry := core.GetEntry()
	entry.Time = record.Time
	entry.Level = slogLevelToCore(record.Level)
	entry.Message = record.Message

	// Add pre-configured attrs
	if len(s.attrs) > 0 {
		entry.Fields = append(entry.Fields, s.attrs...)
	}

	// Add record attrs
	record.Attrs(func(a slog.Attr) bool {
		entry.Fields = append(entry.Fields, slogAttrToField(s.group, a))
		return true
	})

	return s.handler.Handle(entry)
}

// WithAttrs returns a new SlogHandler with additional attributes.
func (s *SlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]core.Field, len(s.attrs), len(s.attrs)+len(attrs))
	copy(newAttrs, s.attrs)
	for _, a := range attrs {
		newAttrs = append(newAttrs, slogAttrToField(s.group, a))
	}
	return &SlogHandler{
		handler: s.handler,
		level:   s.level,
		attrs:   newAttrs,
		group:   s.group,
	}
}

// WithGroup returns a new SlogHandler with the given group name.
func (s *SlogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return s
	}
	newGroup := name
	if s.group != "" {
		newGroup = s.group + "." + name
	}
	newAttrs := make([]core.Field, len(s.attrs))
	copy(newAttrs, s.attrs)
	return &SlogHandler{
		handler: s.handler,
		level:   s.level,
		attrs:   newAttrs,
		group:   newGroup,
	}
}

// slogLevelToCore converts a slog.Level to a core.Level.
func slogLevelToCore(level slog.Level) core.Level {
	switch {
	case level >= slog.LevelError:
		return core.ErrorLevel
	case level >= slog.LevelWarn:
		return core.WarnLevel
	case level >= slog.LevelInfo:
		return core.InfoLevel
	default:
		return core.DebugLevel
	}
}

// slogAttrToField converts a slog.Attr to a core.Field, prepending the group prefix if present.
func slogAttrToField(group string, a slog.Attr) core.Field {
	key := a.Key
	if group != "" {
		key = group + "." + a.Key
	}

	a.Value = a.Value.Resolve()

	switch a.Value.Kind() {
	case slog.KindString:
		return core.Field{Key: key, Type: core.StringType, Str: a.Value.String()}
	case slog.KindInt64:
		return core.Field{Key: key, Type: core.Int64Type, Int64: a.Value.Int64()}
	case slog.KindFloat64:
		return core.Field{Key: key, Type: core.Float64Type, Float64: a.Value.Float64()}
	case slog.KindBool:
		val := int64(0)
		if a.Value.Bool() {
			val = 1
		}
		return core.Field{Key: key, Type: core.BoolType, Int64: val}
	case slog.KindTime:
		t := a.Value.Time()
		return core.Field{Key: key, Type: core.TimeType, Int64: t.UnixNano()}
	case slog.KindDuration:
		return core.Field{Key: key, Type: core.DurationType, Int64: int64(a.Value.Duration())}
	case slog.KindGroup:
		// For group attrs, flatten them with the group prefix
		// This is a simplification - groups become prefixed fields
		attrs := a.Value.Group()
		if len(attrs) > 0 {
			return slogAttrToField(key, attrs[0])
		}
		return core.Field{Key: key, Type: core.AnyType, Any: a.Value.Any()}
	default:
		return core.Field{Key: key, Type: core.AnyType, Any: a.Value.Any()}
	}
}
