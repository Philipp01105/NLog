package logger_test

import (
	"io"

	"github.com/philipp01105/nlog/formatter"
	"github.com/philipp01105/nlog/handler/consolehandler"
	"github.com/philipp01105/nlog/logger"
)

// Use the package-level default logger for quick, no-setup logging.
func Example() {
	logger.Info("Application started")
	logger.Info("User login",
		logger.String("username", "alice"),
		logger.Int("user_id", 123),
	)
}

// Create a custom Logger with the Builder pattern.
func ExampleNewBuilder() {
	ch := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer: io.Discard,
		Async:  false,
		Formatter: formatter.NewTextFormatter(formatter.Config{
			IncludeCaller: true,
		}),
	})

	log := logger.NewBuilder().
		WithHandler(ch).
		WithLevel(logger.DebugLevel).
		WithCaller(true).
		WithFields(logger.String("service", "api")).
		Build()

	log.Info("ready", logger.Int("port", 8080))
	log.Close()
}

// Use With to create a child logger with persistent context fields.
func ExampleLogger_With() {
	ch := consolehandler.NewConsoleHandler(consolehandler.ConsoleConfig{
		Writer: io.Discard,
		Async:  false,
	})

	log := logger.NewBuilder().
		WithHandler(ch).
		Build()

	reqLog := log.With(
		logger.String("request_id", "req-12345"),
		logger.String("method", "GET"),
	)

	reqLog.Info("Processing request", logger.String("path", "/api/users"))
	reqLog.Info("Request completed", logger.Int("status", 200))
	log.Close()
}
