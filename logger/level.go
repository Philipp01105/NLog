package logger

import (
	"strings"

	"github.com/Philipp01105/NLog/core"
)

// Level Re-export type and constants for convenience
type Level = core.Level

const (
	DebugLevel = core.DebugLevel
	InfoLevel  = core.InfoLevel
	WarnLevel  = core.WarnLevel
	ErrorLevel = core.ErrorLevel
	FatalLevel = core.FatalLevel
	PanicLevel = core.PanicLevel
)

// ParseLevel converts a string to a Level
func ParseLevel(s string) Level {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return DebugLevel
	case "INFO":
		return InfoLevel
	case "WARN", "WARNING":
		return WarnLevel
	case "ERROR":
		return ErrorLevel
	case "FATAL":
		return FatalLevel
	case "PANIC":
		return PanicLevel
	default:
		return InfoLevel
	}
}
