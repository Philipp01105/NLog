package formatter

import (
	"bytes"
	"io"
	"strconv"
	"time"

	"github.com/philipp01105/nlog/core"
)

// JSONFormatter formats log entries as JSON
type JSONFormatter struct {
	Config
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter(cfg Config) *JSONFormatter {
	if cfg.TimestampFormat == "" {
		cfg.TimestampFormat = time.RFC3339Nano
	}
	return &JSONFormatter{Config: cfg}
}

// Format formats an entry as JSON
func (f *JSONFormatter) Format(entry *core.Entry) ([]byte, error) {
	buf := getBuffer()
	defer putBuffer(buf)

	f.formatJSONToBuffer(entry, buf)

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// FormatTo formats an entry as JSON and writes it directly to the writer
func (f *JSONFormatter) FormatTo(entry *core.Entry, w io.Writer) error {
	buf := getBuffer()

	f.formatJSONToBuffer(entry, buf)

	_, err := w.Write(buf.Bytes())
	putBuffer(buf)
	return err
}

// FormatEntry formats an entry as JSON into the given buffer (implements BufferFormatter).
func (f *JSONFormatter) FormatEntry(entry *core.Entry, buf *bytes.Buffer) {
	f.formatJSONToBuffer(entry, buf)
}

// formatJSONToBuffer builds JSON manually into the buffer without allocations
func (f *JSONFormatter) formatJSONToBuffer(entry *core.Entry, buf *bytes.Buffer) {
	buf.WriteByte('{')

	// Time field
	buf.WriteString(`"time":"`)
	buf.Write(entry.Time.AppendFormat(buf.AvailableBuffer(), f.TimestampFormat))
	buf.WriteByte('"')

	// Level field
	buf.WriteString(`,"level":"`)
	buf.WriteString(entry.Level.String())
	buf.WriteByte('"')

	// Message field
	buf.WriteString(`,"message":"`)
	appendJSONString(buf, entry.Message)
	buf.WriteByte('"')

	// Caller info if enabled
	if f.IncludeCaller && entry.Caller.Defined {
		buf.WriteString(`,"caller":{"file":"`)
		appendJSONString(buf, entry.Caller.ShortFile)
		buf.WriteString(`","line":`)
		buf.WriteString(strconv.Itoa(entry.Caller.Line))
		if entry.Caller.Function != "" {
			buf.WriteString(`,"function":"`)
			appendJSONString(buf, entry.Caller.Function)
			buf.WriteByte('"')
		}
		buf.WriteByte('}')
	}

	// Fields
	for _, field := range entry.Fields {
		buf.WriteString(`,"`)
		appendJSONString(buf, field.Key)
		buf.WriteString(`":`)
		appendJSONFieldValue(buf, field)
	}

	buf.WriteString("}\n")
}

// appendJSONString writes a JSON-escaped string (without surrounding quotes) to the buffer
func appendJSONString(buf *bytes.Buffer, s string) {
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 0x20 && c != '"' && c != '\\' {
			continue
		}
		// Flush unescaped prefix
		if start < i {
			buf.WriteString(s[start:i])
		}
		switch c {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			buf.WriteString(`\u00`)
			buf.WriteByte(hexChars[c>>4])
			buf.WriteByte(hexChars[c&0x0f])
		}
		start = i + 1
	}
	// Flush remaining
	if start < len(s) {
		buf.WriteString(s[start:])
	}
}

var hexChars = [16]byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f'}

// appendJSONFieldValue writes a JSON-encoded field value to the buffer
func appendJSONFieldValue(buf *bytes.Buffer, field core.Field) {
	switch field.Type {
	case core.StringType:
		buf.WriteByte('"')
		appendJSONString(buf, field.Str)
		buf.WriteByte('"')
	case core.IntType, core.Int64Type:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), field.Int64, 10))
	case core.Float64Type:
		buf.Write(strconv.AppendFloat(buf.AvailableBuffer(), field.Float64, 'f', -1, 64))
	case core.BoolType:
		buf.Write(strconv.AppendBool(buf.AvailableBuffer(), field.Int64 == 1))
	case core.TimeType:
		buf.WriteByte('"')
		buf.Write(time.Unix(0, field.Int64).AppendFormat(buf.AvailableBuffer(), time.RFC3339Nano))
		buf.WriteByte('"')
	case core.DurationType:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), field.Int64, 10))
	case core.ErrorType:
		buf.WriteByte('"')
		appendJSONString(buf, field.Str)
		buf.WriteByte('"')
	default:
		buf.WriteByte('"')
		appendJSONString(buf, field.StringValue())
		buf.WriteByte('"')
	}
}
