// Package formatter defines how log entries are serialized into bytes.
//
// It exposes two interfaces: Formatter, which returns a []byte, and
// WriterFormatter, which writes directly to an io.Writer. Handlers
// check for WriterFormatter at construction time and prefer it when
// available, eliminating the intermediate byte slice allocation on
// the write path.
//
// Both built-in formatters (TextFormatter and JSONFormatter) implement
// both interfaces. They use a pooled bytes.Buffer internally and rely
// on Go's Append-style functions (time.AppendFormat, strconv.AppendInt)
// to avoid per-call allocations. The TextFormatter additionally
// pre-computes level bracket strings (" [INFO] ", etc.) so that the
// most common path is a single WriteString call.
//
// Buffers larger than 64 KiB are not returned to the pool to prevent
// a single large log line from permanently inflating memory usage.
package formatter
