# Go Logging Framework

A lightweight, high-performance logging framework for Go with zero-allocation optimizations and flexible configuration.

## Features

✅ **Immutable Logger Instances** - Built using the Builder pattern for thread-safe, immutable configuration  
✅ **Async Logging** - Handler-specific async option with configurable buffer size (opt-out per handler, async by default)  
✅ **Configurable Overflow Policies** - Choose behavior when queue is full: DropNewest, DropOldest, or Block with timeout  
✅ **Telemetry & Observability** - Track dropped logs, processed logs, and blocked writes per handler  
✅ **Configurable Caller Info** - Optional file/line/function tracking  
✅ **Advanced File Rotation** - Rotate by size, age, interval, with configurable backup retention  
✅ **Multiple Formatters** - Text and JSON formatters with zero-copy WriterFormatter support  
✅ **Structured Logging** - Type-safe fields for structured logs  
✅ **Zero-Allocation Logging** - 0 allocs/op for Info calls, entry pooling, level-check before allocation  
✅ **Fatal & Panic Levels** - Fatal calls `os.Exit(1)`, Panic triggers `panic()` after logging  
✅ **`log/slog` Compatibility** - Drop-in slog.Handler adapter for standard library integration  
✅ **Slow-IO Resilient** - Bounded memory, observable behavior, block with sync fallback for critical logs  
✅ **Multi-Handler Support** - Send logs to multiple outputs simultaneously  
✅ **Shutdown Correctness** - Idempotent Close() with timeout-based queue draining  

## Installation

```bash
go get github.com/Philipp01105/logging-framework
```

Requires Go 1.22 or later.

## Quick Start

```go
package main

import (
	"github.com/Philipp01105/logging-framework/logger"
)

func main() {
	// Use the default logger (async console handler, Info level)
	logger.Info("Application started")
	logger.Info("User login", 
		logger.String("username", "alice"),
		logger.Int("user_id", 123),
	)
}
```

## Configuration

### Custom Logger with Builder Pattern

```go
package main

import (
	"github.com/Philipp01105/logging-framework/formatter"
	"github.com/Philipp01105/logging-framework/handler"
	"github.com/Philipp01105/logging-framework/logger"
)

func main() {
	// Create a custom console handler
	consoleHandler := handler.NewConsoleHandler(handler.ConsoleConfig{
		Async:      true,
		BufferSize: 1000,
		Formatter:  formatter.NewTextFormatter(formatter.Config{
			IncludeCaller: true,
		}),
	})

	// Build an immutable logger
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

### File Handler with Rotation

```go
fileHandler, err := handler.NewFileHandler(handler.FileConfig{
	Filename:   "/var/log/app.log",
	Async:      true,
	BufferSize: 1000,
	MaxSize:    10 * 1024 * 1024, // 10MB
	MaxAge:     24 * time.Hour,     // Rotate after 24 hours
	Formatter:  formatter.NewTextFormatter(formatter.Config{}),
})

if err != nil {
	panic(err)
}
defer fileHandler.Close()

fileLogger := logger.NewBuilder().
	WithHandler(fileHandler).
	WithLevel(logger.InfoLevel).
	Build()

fileLogger.Info("Log entry written to file")
```

### JSON Formatter

```go
jsonHandler := handler.NewConsoleHandler(handler.ConsoleConfig{
	Formatter: formatter.NewJSONFormatter(formatter.Config{
		IncludeCaller: false,
	}),
})

jsonLogger := logger.NewBuilder().
	WithHandler(jsonHandler).
	WithLevel(logger.InfoLevel).
	Build()

jsonLogger.Info("JSON formatted log",
	logger.String("service", "api"),
	logger.Float64("response_time", 0.123),
	logger.Time("timestamp", time.Now()),
)
// Output: {"level":"INFO","message":"JSON formatted log","response_time":0.123,"service":"api","time":"2026-02-18T13:00:00Z","timestamp":"2026-02-18T13:00:00Z"}
```

### Multi-Handler (Console + File)

```go
consoleHandler := handler.NewConsoleHandler(handler.ConsoleConfig{
	Async: true,
})

fileHandler, _ := handler.NewFileHandler(handler.FileConfig{
	Filename: "/var/log/app.log",
	Async:    true,
})

multiHandler := handler.NewMultiHandler(consoleHandler, fileHandler)

logger := logger.NewBuilder().
	WithHandler(multiHandler).
	WithLevel(logger.InfoLevel).
	Build()

logger.Info("This goes to both console and file")
```

### Synchronous Logging (Opt-out of Async)

```go
syncHandler := handler.NewConsoleHandler(handler.ConsoleConfig{
	Async: false, // Disable async for synchronous logging
})

syncLogger := logger.NewBuilder().
	WithHandler(syncHandler).
	WithLevel(logger.InfoLevel).
	Build()

syncLogger.Info("Synchronous log entry")
```

## Log Levels

The framework supports six log levels (in order of severity):

- `DebugLevel` - Detailed debugging information
- `InfoLevel` - General informational messages (default)
- `WarnLevel` - Warning messages
- `ErrorLevel` - Error messages
- `FatalLevel` - Fatal errors; logs the message and calls `os.Exit(1)`
- `PanicLevel` - Panic errors; logs the message and calls `panic()`

Logs are only processed if their level is equal to or higher than the logger's configured level.

## Structured Fields

The framework provides type-safe field constructors:

```go
logger.Info("User action",
	logger.String("username", "alice"),
	logger.Int("age", 30),
	logger.Int64("user_id", 123456789),
	logger.Float64("score", 98.5),
	logger.Bool("admin", true),
	logger.Time("login_time", time.Now()),
	logger.Duration("elapsed", 5*time.Second),
	logger.Err(err),                          // For errors
	logger.Any("custom", customObject),       // For any type
)
```

## Contextual Logging with `With()`

Create child loggers with additional context fields (immutable operation):

```go
requestLogger := logger.With(
	logger.String("request_id", "req-12345"),
	logger.String("method", "GET"),
)

requestLogger.Info("Processing request", logger.String("path", "/api/users"))
requestLogger.Info("Request completed", logger.Int("status", 200))
```

## Performance

The framework is designed for high performance with:

- **Zero-Allocation Info Path**: 0 allocs/op for `Info()` calls with entry pooling and WriterFormatter
- **Level Check Optimization**: Exits early if log level is too low (~8 ns, 0 allocs)
- **Buffer Pooling**: Reuses buffers for formatting via WriterFormatter zero-copy interface
- **Async Processing**: Non-blocking log writes (async by default)
- **Configurable Overflow Policies**: DropNewest, DropOldest, or Block strategies

### Benchmark Results

```
BenchmarkInfoNoFields              144 ns/op       0 B/op    0 allocs/op
BenchmarkInfoWith2Fields           202 ns/op       0 B/op    0 allocs/op
BenchmarkJSON                      445 ns/op       0 B/op    0 allocs/op
BenchmarkZeroAllocFiltered         0.3 ns/op       0 B/op    0 allocs/op
BenchmarkFilteredDebug             8.8 ns/op       0 B/op    0 allocs/op
BenchmarkMultiGoroutineContention   29 ns/op       5 B/op    0 allocs/op
BenchmarkQueueFullStress            18 ns/op       0 B/op    0 allocs/op
```

## Architecture

### Package Structure

- `core/` - Core types (Entry, Field, Level) shared across packages
- `logger/` - Main Logger API, Builder, and convenience functions
- `handler/` - Handler interface and implementations (Console, File, Multi, SlogHandler)
- `formatter/` - Formatter interface and implementations (Text, JSON, WriterFormatter)
- `examples/` - Example programs (slog compatibility, advanced features demo)

### Design Patterns

- **Builder Pattern**: For constructing immutable Logger instances
- **Strategy Pattern**: Pluggable handlers and formatters
- **Object Pool Pattern**: For zero-allocation Entry reuse

## Testing

Run tests:
```bash
go test ./...
```

Run benchmarks:
```bash
go test -bench=. -benchmem ./...
```

## Advanced Features

### `log/slog` Compatibility

The framework provides a drop-in `slog.Handler` adapter for seamless integration with Go's standard `log/slog` package:

```go
package main

import (
	"log/slog"

	"github.com/Philipp01105/logging-framework/core"
	"github.com/Philipp01105/logging-framework/formatter"
	"github.com/Philipp01105/logging-framework/handler"
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

### Overflow Policies

Control what happens when async queues fill up:

```go
import "github.com/Philipp01105/logging-framework/handler"

h := handler.NewConsoleHandler(handler.ConsoleConfig{
    OverflowPolicy: map[core.Level]handler.OverflowPolicy{
        core.DebugLevel: handler.DropNewest,  // Drop new entries
        core.ErrorLevel: handler.Block,       // Block with timeout
    },
    BlockTimeout: 100 * time.Millisecond,
    DrainTimeout: 5 * time.Second,
})
```

**Available policies:**
- `DropNewest`: Drop incoming log (default for DEBUG/INFO/WARN)
- `DropOldest`: Remove oldest from queue
- `Block`: Block caller with timeout, fallback to sync write (default for ERROR)

### Telemetry

Monitor logging behavior in real-time:

```go
stats := handler.Stats()
fmt.Printf("Processed: %d\n", stats.ProcessedTotal)
fmt.Printf("Dropped: %d\n", stats.DroppedTotal[core.InfoLevel])
fmt.Printf("Blocked: %d\n", stats.BlockedTotal)
```

Use these metrics to detect slow I/O, tune buffer sizes, and monitor application load.

### Advanced File Rotation

Multiple rotation triggers and backup management:

```go
handler.NewFileHandler(handler.FileConfig{
    Filename:       "/var/log/app.log",
    MaxSize:        100 * 1024 * 1024,  // Rotate at 100MB
    MaxAge:         7 * 24 * time.Hour,  // Rotate after 7 days
    RotateInterval: 24 * time.Hour,      // Also rotate daily
    MaxBackups:     30,                   // Keep 30 old files
})
```

### Zero-Allocation Path

Filtered logs cause **zero allocations**:

```go
logger := logger.NewBuilder().
    WithLevel(logger.InfoLevel).
    Build()

logger.Debug("filtered") // 0.3 ns/op, 0 allocs
```

The level check happens *before* `GetEntry()`, ensuring true zero-allocation for filtered logs.

### Slow-IO Resilience

The framework handles slow disk I/O gracefully:

- **Bounded memory**: Queue size limits prevent unbounded growth
- **Observable**: Track drops and blocks via telemetry
- **Fallback**: Block policy falls back to sync write on timeout

See [ADVANCED.md](ADVANCED.md) for detailed documentation on all advanced features.

## License

This project is open source and available under the MIT License.
