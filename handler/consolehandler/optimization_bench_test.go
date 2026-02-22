package consolehandler

import (
	"io"
	"testing"

	"github.com/philipp01105/nlog/core"
	"github.com/philipp01105/nlog/formatter"
)

// BenchmarkConsoleHandler_WriterFormatter benchmarks ConsoleHandler with WriterFormatter (zero-alloc path)
func BenchmarkConsoleHandler_WriterFormatter(b *testing.B) {
	ch := NewConsoleHandler(ConsoleConfig{
		Writer:    io.Discard,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})
	defer ch.Close()

	h := ch.(*SyncConsoleHandler)
	entry := core.GetEntry()
	entry.Level = core.InfoLevel
	entry.Message = "benchmark message"
	entry.Fields = []core.Field{
		{Key: "key1", Type: core.StringType, Str: "value1"},
		{Key: "key2", Type: core.Int64Type, Int64: 42},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		h.write(entry, &h.parBufPool)
	}
}
