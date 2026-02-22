package consolehandler_test

import (
	"io"

	"github.com/philipp01105/nlog/formatter"
	"github.com/philipp01105/nlog/handler/consolehandler"
)

// Create a synchronous console handler writing to stdout.
func ExampleNewConsoleHandler() {
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer: io.Discard,
		Async:  false,
		Formatter: formatter.NewTextFormatter(formatter.Config{
			IncludeCaller: false,
		}),
	})
	defer h.Close()
}

// Create an async console handler with a custom buffer size.
func ExampleNewConsoleHandler_async() {
	h := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:     io.Discard,
		Async:      true,
		BufferSize: 4096,
		Formatter:  formatter.NewJSONFormatter(formatter.Config{}),
	})
	defer h.Close()
}
