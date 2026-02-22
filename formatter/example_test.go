package formatter_test

import (
	"fmt"
	"strings"
	"time"

	"github.com/philipp01105/nlog/core"
	"github.com/philipp01105/nlog/formatter"
)

func ExampleNewTextFormatter() {
	f := formatter.NewTextFormatter(formatter.Config{})

	entry := &core.Entry{
		Time:    time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
		Level:   core.InfoLevel,
		Message: "hello world",
	}

	out, _ := f.Format(entry)
	// Timestamp prefix followed by level and message.
	fmt.Println(strings.Contains(string(out), "[INFO]"))
	fmt.Println(strings.Contains(string(out), "hello world"))
	// Output:
	// true
	// true
}

func ExampleNewJSONFormatter() {
	f := formatter.NewJSONFormatter(formatter.Config{})

	entry := &core.Entry{
		Time:    time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
		Level:   core.InfoLevel,
		Message: "request handled",
		Fields: []core.Field{
			{Key: "status", Int64: 200, Type: core.Int64Type},
		},
	}

	out, _ := f.Format(entry)
	fmt.Println(strings.Contains(string(out), `"level":"INFO"`))
	fmt.Println(strings.Contains(string(out), `"message":"request handled"`))
	// Output:
	// true
	// true
}
