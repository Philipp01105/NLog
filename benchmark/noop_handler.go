package benchmark

import (
	"github.com/philipp01105/nlog/core"
	"github.com/philipp01105/nlog/handler"
)

type noopHandler struct{}

func newNoopHandler() handler.Handler {
	return &noopHandler{}
}

func (h *noopHandler) Handle(e *core.Entry) error {
	_ = len(e.Message)
	core.PutEntry(e)
	return nil
}

func (h *noopHandler) Close() error {
	return nil
}
