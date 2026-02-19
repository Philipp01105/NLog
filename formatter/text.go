package formatter

import (
	"bytes"
	"io"
	"strconv"
	"time"

	"github.com/philipp01105/nlog/core"
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
	if f.IncludeCaller && entry.Caller.Defined {
		buf.WriteByte('[')
		buf.WriteString(entry.Caller.ShortFile)
		buf.WriteByte(':')
		buf.WriteString(strconv.Itoa(entry.Caller.Line))
		buf.WriteString("] ")
	}

	// Message
	buf.WriteString(entry.Message)

	// Fields - write values directly to buffer to avoid intermediate string allocations
	for _, field := range entry.Fields {
		buf.WriteByte(' ')
		buf.WriteString(field.Key)
		buf.WriteByte('=')
		appendTextFieldValue(buf, field)
	}

	buf.WriteByte('\n')
}

// appendTextFieldValue writes a field value directly to the buffer without intermediate string allocation
func appendTextFieldValue(buf *bytes.Buffer, field core.Field) {
	switch field.Type {
	case core.StringType:
		buf.WriteString(field.Str)
	case core.IntType, core.Int64Type:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), field.Int64, 10))
	case core.Float64Type:
		buf.Write(strconv.AppendFloat(buf.AvailableBuffer(), field.Float64, 'f', -1, 64))
	case core.BoolType:
		buf.Write(strconv.AppendBool(buf.AvailableBuffer(), field.Int64 == 1))
	case core.TimeType:
		buf.Write(time.Unix(0, field.Int64).AppendFormat(buf.AvailableBuffer(), time.RFC3339))
	case core.DurationType:
		buf.WriteString(time.Duration(field.Int64).String())
	case core.ErrorType:
		buf.WriteString(field.Str)
	default:
		buf.WriteString(field.StringValue())
	}
}
