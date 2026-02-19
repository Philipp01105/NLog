package core

import (
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Level represents the severity level of a log entry
type Level int8

const (
	// DebugLevel for detailed debugging information
	DebugLevel Level = iota
	// InfoLevel for general informational messages (default)
	InfoLevel
	// WarnLevel for warning messages
	WarnLevel
	// ErrorLevel for error messages
	ErrorLevel
	// FatalLevel for fatal messages (causes os.Exit(1))
	FatalLevel
	// PanicLevel for panic messages (causes panic)
	PanicLevel
)

// String returns the string representation of the level
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	case PanicLevel:
		return "PANIC"
	default:
		return "UNKNOWN"
	}
}

// Entry represents a log entry with all its metadata
type Entry struct {
	Time    time.Time
	Level   Level
	Message string
	Fields  []Field
	Caller  CallerInfo
}

// CallerInfo contains information about the caller
type CallerInfo struct {
	File      string
	ShortFile string
	Line      int
	Function  string
	Defined   bool
}

// entryPool is a pool of Entry objects to reduce allocations
var entryPool = sync.Pool{
	New: func() interface{} {
		return &Entry{
			Fields: make([]Field, 0, 8), // Pre-allocate for 8 fields
		}
	},
}

// GetEntry retrieves an Entry from the pool
func GetEntry() *Entry {
	e := entryPool.Get().(*Entry)
	e.Time = time.Now()
	e.Fields = e.Fields[:0]
	e.Caller = CallerInfo{}
	return e
}

// PutEntry returns an Entry to the pool
func PutEntry(e *Entry) {
	if e == nil {
		return
	}
	// Re-slice to zero length; GC handles reference cleanup
	e.Fields = e.Fields[:0]
	e.Message = ""
	e.Caller = CallerInfo{}
	entryPool.Put(e)
}

// GetCaller retrieves caller information
func GetCaller(skip int) CallerInfo {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return CallerInfo{}
	}

	fn := runtime.FuncForPC(pc)
	var funcName string
	if fn != nil {
		funcName = fn.Name()
	}

	return CallerInfo{
		File:      file,
		ShortFile: filepath.Base(file),
		Line:      line,
		Function:  funcName,
		Defined:   true,
	}
}
