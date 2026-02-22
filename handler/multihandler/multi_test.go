package multihandler

import (
	"bytes"
	"strings"
	"testing"

	"github.com/philipp01105/nlog/core"
	"github.com/philipp01105/nlog/formatter"
	"github.com/philipp01105/nlog/handler/consolehandler"
)

func TestMultiHandler(t *testing.T) {
	var buf1, buf2 bytes.Buffer

	h1 := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    &buf1,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})

	h2 := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer:    &buf2,
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})

	multi := NewMultiHandler(h1, h2)
	defer multi.Close()

	entry := core.GetEntry()
	entry.Level = core.InfoLevel
	entry.Message = "multi test"

	err := multi.Handle(entry)
	if err != nil {
		t.Errorf("Handle() error = %v", err)
	}

	if !strings.Contains(buf1.String(), "multi test") {
		t.Error("First handler did not receive message")
	}

	if !strings.Contains(buf2.String(), "multi test") {
		t.Error("Second handler did not receive message")
	}
}
