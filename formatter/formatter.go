package formatter

import (
	"bytes"
	"io"
	"sync"

	"github.com/philipp01105/nlog/core"
)

// Formatter defines the interface for log formatters
type Formatter interface {
	// Format formats a log entry into bytes
	Format(entry *core.Entry) ([]byte, error)
}

// WriterFormatter is an optional interface that formatters can implement
// to write directly to a writer without intermediate byte slice allocation.
type WriterFormatter interface {
	// FormatTo formats a log entry and writes it directly to the writer
	FormatTo(entry *core.Entry, w io.Writer) error
}

// BufferFormatter is an optional interface that formatters can implement
// to format directly into a caller-provided buffer, avoiding internal
// buffer pool overhead.
type BufferFormatter interface {
	// FormatEntry formats a log entry into the given buffer.
	FormatEntry(entry *core.Entry, buf *bytes.Buffer)
}

// Config holds common formatter configuration
type Config struct {
	// IncludeCaller enables caller information in log output
	IncludeCaller bool
	// TimestampFormat specifies the time format (empty for RFC3339)
	TimestampFormat string
}

// bufferPool is a pool of bytes.Buffer to reduce allocations
var bufferPool = &sync.Pool{
	New: func() interface{} {
		b := new(bytes.Buffer)
		b.Grow(256)
		return b
	},
}

func getBuffer() *bytes.Buffer {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func putBuffer(buf *bytes.Buffer) {
	if buf.Cap() > 64*1024 { // Don't keep very large buffers
		return
	}
	bufferPool.Put(buf)
}
