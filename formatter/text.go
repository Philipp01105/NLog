package formatter

import (
	"bytes"
	"io"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Philipp01105/logging-framework/core"
)

// TextFormatter formats log entries as human-readable text
type TextFormatter struct {
	Config
}

// NewTextFormatter creates a new text formatter
func NewTextFormatter(cfg Config) *TextFormatter {
	if cfg.TimestampFormat == "" {
		cfg.TimestampFormat = time.RFC3339
	}
	return &TextFormatter{Config: cfg}
}

// Format formats an entry as text
func (f *TextFormatter) Format(entry *core.Entry) ([]byte, error) {
	buf := getBuffer()
	defer putBuffer(buf)

	f.formatToBuffer(entry, buf)

	// Copy buffer content to return
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// FormatTo formats an entry and writes it directly to the writer
func (f *TextFormatter) FormatTo(entry *core.Entry, w io.Writer) error {
	buf := getBuffer()

	f.formatToBuffer(entry, buf)

	_, err := w.Write(buf.Bytes())
	putBuffer(buf)
	return err
}

// pre-formatted level strings to avoid multiple WriteString calls
var levelBrackets = [...]string{
	core.DebugLevel: " [DEBUG] ",
	core.InfoLevel:  " [INFO] ",
	core.WarnLevel:  " [WARN] ",
	core.ErrorLevel: " [ERROR] ",
	core.FatalLevel: " [FATAL] ",
	core.PanicLevel: " [PANIC] ",
}

// formatToBuffer writes the formatted entry into the given buffer
func (f *TextFormatter) formatToBuffer(entry *core.Entry, buf *bytes.Buffer) {
	// Timestamp - use AppendFormat to avoid string allocation
	buf.Write(entry.Time.AppendFormat(buf.AvailableBuffer(), f.TimestampFormat))

	// Level - use pre-formatted string
	if int(entry.Level) < len(levelBrackets) {
		buf.WriteString(levelBrackets[entry.Level])
	} else {
		buf.WriteString(" [UNKNOWN] ")
	}

	// Caller info if enabled
	if f.IncludeCaller && entry.Caller != nil {
		file := filepath.Base(entry.Caller.File)
		buf.WriteByte('[')
		buf.WriteString(file)
		buf.WriteByte(':')
		buf.WriteString(strconv.Itoa(entry.Caller.Line))
		buf.WriteString("] ")
	}

	// Message
	buf.WriteString(entry.Message)

	// Fields
	for _, field := range entry.Fields {
		buf.WriteByte(' ')
		buf.WriteString(field.Key)
		buf.WriteByte('=')
		buf.WriteString(field.StringValue())
	}

	buf.WriteByte('\n')
}
