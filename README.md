# Go Logging Framework

A lightweight, high-performance structured logging framework for Go with zero-allocation optimizations, async processing, and flexible configuration.

The framework is designed around **immutable logger instances** using the Builder pattern, making it inherently thread-safe. Logging is **async by default** with configurable overflow policies, and the hot path achieves **zero allocations**.

## Installation

```bash
go get github.com/philipp01105/NLog
```

Requires Go 1.22 or later.

#### Example

The simplest way to use the framework is the package-level default logger:

```go
package main

import (
	"github.com/philipp01105/NLog/logger"
)

func main() {
	logger.Info("Application started")
	logger.Info("User login",
		logger.String("username", "alice"),
		logger.Int("user_id", 123),
	)
}
```

For more advanced usage, create a custom logger instance with the Builder:

```go
package main

import (
	"github.com/philipp01105/NLog/formatter"
	"github.com/philipp01105/NLog/handler"
	"github.com/philipp01105/NLog/logger"
)

func main() {
	consoleHandler := handler.NewConsoleHandler(handler.ConsoleConfig{
		Async:      true,
		BufferSize: 1000,
		Formatter: formatter.NewTextFormatter(formatter.Config{
			IncludeCaller: true,
		}),
	})

	myLogger := logger.NewBuilder().
		WithHandler(consoleHandler).
		WithLevel(logger.DebugLevel).
		WithCaller(true).
		WithFields(logger.String("service", "api")).
		Build()

	myLogger.Debug("Debug message with caller info")
	myLogger.Info("Info message", logger.Int("count", 42))
}
```

With `formatter.NewJSONFormatter`, for easy parsing by logstash or Splunk:

```text
{"level":"INFO","message":"JSON formatted log","response_time":0.123,"service":"api","time":"2026-02-18T13:00:00Z","timestamp":"2026-02-18T13:00:00Z"}
```

#### Fields

The framework encourages structured logging through type-safe field constructors instead of format strings:

```go
logger.Info("User action",
	logger.String("username", "alice"),
	logger.Int("age", 30),
	logger.Int64("user_id", 123456789),
	logger.Float64("score", 98.5),
	logger.Bool("admin", true),
	logger.Time("login_time", time.Now()),
	logger.Duration("elapsed", 5*time.Second),
	logger.Err(err),
	logger.Any("custom", customObject),
)
```

#### Default Fields

Often it's helpful to have fields _always_ attached to log statements in an application or parts of one. Instead of repeating fields on every line, use `With()` to create a child logger with persistent context (immutable operation):

```go
requestLogger := logger.With(
	logger.String("request_id", "req-12345"),
	logger.String("method", "GET"),
)

requestLogger.Info("Processing request", logger.String("path", "/api/users"))
requestLogger.Info("Request completed", logger.Int("status", 200))
```

#### Logging Method Name

If you wish to add the calling method as a field, enable caller reporting:

```go
myLogger := logger.NewBuilder().
	WithCaller(true).
	Build()
```

Note that this does add measurable overhead.

#### Level Logging

The framework has six logging levels: Debug, Info, Warning, Error, Fatal and Panic.

```go
logger.Debug("Useful debugging information.")
logger.Info("Something noteworthy happened!")
logger.Warn("You should probably take a look at this.")
logger.Error("Something failed but I'm not quitting.")
// Calls os.Exit(1) after logging
logger.Fatal("Bye.")
// Calls panic() after logging
logger.Panic("I'm bailing.")
```

You can set the logging level on a Logger, then it will only log entries with that severity or anything above it:

```go
myLogger := logger.NewBuilder().
	WithLevel(logger.InfoLevel). // Default. Will log Info and above.
	Build()
```

#### Formatters

The built-in logging formatters are:

* `formatter.NewTextFormatter` — Human-readable text output with optional caller info.
* `formatter.NewJSONFormatter` — Logs fields as JSON.

Both formatters support the zero-copy `WriterFormatter` interface for zero-allocation formatting.

You can define your own formatter by implementing the `Formatter` interface.

#### Handlers

The framework ships with multiple handler implementations:

* **ConsoleHandler** — Writes to stdout/stderr. Async by default.
* **FileHandler** — Writes to files with built-in rotation (by size, age, or interval).
* **MultiHandler** — Fan-out to multiple handlers simultaneously.
* **SlogHandler** — Drop-in `slog.Handler` adapter for `log/slog` compatibility.

```go
consoleHandler := handler.NewConsoleHandler(handler.ConsoleConfig{Async: true})

fileHandler, _ := handler.NewFileHandler(handler.FileConfig{
	Filename: "/var/log/app.log",
	Async:    true,
})

multiHandler := handler.NewMultiHandler(consoleHandler, fileHandler)

myLogger := logger.NewBuilder().
	WithHandler(multiHandler).
	Build()

myLogger.Info("This goes to both console and file")
```

#### Synchronous Logging

Async is the default. To opt out, disable it per handler:

```go
syncHandler := handler.NewConsoleHandler(handler.ConsoleConfig{
	Async: false,
})
```

#### File Rotation

Unlike many logging libraries, file rotation is built-in. Multiple rotation triggers and backup management are supported:

```go
handler.NewFileHandler(handler.FileConfig{
	Filename:       "/var/log/app.log",
	MaxSize:        100 * 1024 * 1024,   // Rotate at 100MB
	MaxAge:         7 * 24 * time.Hour,   // Rotate after 7 days
	RotateInterval: 24 * time.Hour,       // Also rotate daily
	MaxBackups:     30,                    // Keep 30 old files
})
```

#### Overflow Policies

Control what happens when async queues fill up:

```go
h := handler.NewConsoleHandler(handler.ConsoleConfig{
	OverflowPolicy: map[core.Level]handler.OverflowPolicy{
		core.DebugLevel: handler.DropNewest,
		core.ErrorLevel: handler.Block,
	},
	BlockTimeout: 100 * time.Millisecond,
	DrainTimeout: 5 * time.Second,
})
```

Available policies:

* `DropNewest` — Drop incoming log entry (default for DEBUG/INFO/WARN)
* `DropOldest` — Remove oldest entry from queue
* `Block` — Block caller with timeout, fallback to sync write (default for ERROR)

#### Telemetry

Monitor logging behavior in real-time to detect slow I/O, tune buffer sizes, and observe application load:

```go
stats := handler.Stats()
fmt.Printf("Processed: %d\n", stats.ProcessedTotal)
fmt.Printf("Dropped: %d\n", stats.DroppedTotal[core.InfoLevel])
fmt.Printf("Blocked: %d\n", stats.BlockedTotal)
```

#### `log/slog` Compatibility

The framework provides a drop-in `slog.Handler` adapter for seamless integration with Go's standard `log/slog` package:

```go
package main

import (
	"log/slog"

	"github.com/Philipp01105/NLog/core"
	"github.com/Philipp01105/NLog/formatter"
	"github.com/Philipp01105/NLog/handler"
)

func main() {
	consoleHandler := handler.NewConsoleHandler(handler.ConsoleConfig{
		Async:     false,
		Formatter: formatter.NewTextFormatter(formatter.Config{}),
	})

	slogHandler := handler.NewSlogHandler(consoleHandler, core.InfoLevel)
	logger := slog.New(slogHandler)

	logger.Info("Hello from slog", "user", "alice", "count", 42)
	logger.Warn("Something might be wrong", "component", "auth")

	consoleHandler.Close()
}
```

#### Thread Safety

Logger instances are immutable after construction via the Builder pattern, making them inherently safe for concurrent use. Async handlers use bounded queues with configurable overflow policies. Shutdown is handled via an idempotent `Close()` with timeout-based queue draining.

#### Performance

The framework achieves zero allocations on the hot path through entry pooling, level-check-before-allocation, and the `WriterFormatter` zero-copy interface.

```text
BenchmarkInfoNoFields              144 ns/op       0 B/op    0 allocs/op
BenchmarkInfoWith2Fields           202 ns/op       0 B/op    0 allocs/op
BenchmarkJSON                      445 ns/op       0 B/op    0 allocs/op
BenchmarkZeroAllocFiltered         0.3 ns/op       0 B/op    0 allocs/op
BenchmarkFilteredDebug             8.8 ns/op       0 B/op    0 allocs/op
BenchmarkMultiGoroutineContention   29 ns/op       5 B/op    0 allocs/op
BenchmarkQueueFullStress            18 ns/op       0 B/op    0 allocs/op
```

Filtered logs (level too low) cause **zero allocations** — the level check happens *before* `GetEntry()`:

```go
myLogger := logger.NewBuilder().
	WithLevel(logger.InfoLevel).
	Build()

myLogger.Debug("filtered") // 0.3 ns/op, 0 allocs
```

#### Package Structure

| Package | Description |
| ------- | ----------- |
| `core/` | Core types (Entry, Field, Level) shared across packages |
| `logger/` | Main Logger API, Builder, and convenience functions |
| `handler/` | Handler interface and implementations (Console, File, Multi, SlogHandler) |
| `formatter/` | Formatter interface and implementations (Text, JSON, WriterFormatter) |

#### Testing

Run tests:

```bash
go test ./...
```

Run benchmarks:

```bash
go test -bench=. -benchmem ./...
```

#### License

This project is open source and available under the MIT License.