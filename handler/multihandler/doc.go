// Package multihandler provides a fan-out handler that dispatches log
// entries to multiple child handlers simultaneously.
//
// When all children implement FastHandler, MultiHandler also satisfies
// FastHandler and avoids entry allocation entirely. Use
// NewMultiHandler to create a handler that fans out to any number of
// child handlers:
//
//	multi := multihandler.NewMultiHandler(consoleH, fileH)
//
// Close calls are forwarded to all children.
package multihandler
