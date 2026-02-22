package consolehandler

import (
	"bytes"
	"io"
	"os"
	"sync"
	"time"

	"github.com/philipp01105/nlog/core"
	"github.com/philipp01105/nlog/formatter"
	"github.com/philipp01105/nlog/handler"
)

// lockedWriter wraps an io.Writer with a mutex, acquiring the lock only
// for Write calls. Formatters prepare data in their own pooled buffers
// and call Write once, so the lock is held only during the actual I/O.
// Uses the handler's main mu to serialize all writes.
type lockedWriter struct {
	mu *sync.Mutex // points to handler's mu
	w  io.Writer
}

func (lw *lockedWriter) Write(p []byte) (n int, err error) {
	lw.mu.Lock()
	n, err = lw.w.Write(p)
	lw.mu.Unlock()
	return
}

// isConcurrentSafeWriter returns true if the writer is known to be safe for
// concurrent Write calls, allowing the handler to skip write-level locking.
func isConcurrentSafeWriter(w io.Writer) bool {
	if w == io.Discard {
		return true
	}
	_, ok := w.(*os.File)
	return ok
}

// consoleBase contains shared fields and methods for console handlers.
type consoleBase struct {
	writer          io.Writer
	formatter       formatter.Formatter
	writerFormatter formatter.WriterFormatter
	bufferFormatter formatter.BufferFormatter
	concurrentSafe  bool // true if writer is safe for concurrent Write calls
	stats           *handler.Stats
	mu              sync.Mutex // protects syncBuf and writer (single lock)
	lw              lockedWriter
	syncBuf         bytes.Buffer
	closed          chan struct{}
}

// write formats and writes an entry.
// Uses TryLock on mu to access handler-owned buffer when uncontended (zero pool
// overhead). When contended and bufferFormatter is available, uses the provided
// parBufPool to format outside the lock, then writes under mu. Otherwise, falls
// through to writerFormatter or generic formatter paths.
func (b *consoleBase) write(entry *core.Entry, parBufPool *sync.Pool) error {
	if b.bufferFormatter != nil {
		if b.mu.TryLock() {
			b.syncBuf.Reset()
			b.bufferFormatter.FormatEntry(entry, &b.syncBuf)
			_, err := b.writer.Write(b.syncBuf.Bytes())
			b.mu.Unlock()
			if err == nil {
				b.stats.IncrementProcessed()
			}
			return err
		}

		// Parallel fallback: format in pool buffer outside lock, then
		// write under mu (or directly for concurrent-safe writers).
		pb := parBufPool.Get().(*parallelBuf)
		pb.buf.Reset()
		b.bufferFormatter.FormatEntry(entry, &pb.buf)
		var err error
		if b.concurrentSafe {
			_, err = b.writer.Write(pb.buf.Bytes())
		} else {
			b.mu.Lock()
			_, err = b.writer.Write(pb.buf.Bytes())
			b.mu.Unlock()
		}
		if err == nil {
			b.stats.IncrementProcessed()
		}
		parBufPool.Put(pb)
		return err
	}

	if b.writerFormatter != nil {
		var err error
		if b.concurrentSafe {
			err = b.writerFormatter.FormatTo(entry, b.writer)
		} else {
			err = b.writerFormatter.FormatTo(entry, &b.lw)
		}
		if err == nil {
			b.stats.IncrementProcessed()
		}
		return err
	}

	data, err := b.formatter.Format(entry)
	if err != nil {
		return err
	}

	if b.concurrentSafe {
		_, writeErr := b.writer.Write(data)
		if writeErr == nil {
			b.stats.IncrementProcessed()
		}
		return writeErr
	}

	b.mu.Lock()
	_, writeErr := b.writer.Write(data)
	b.mu.Unlock()

	if writeErr == nil {
		b.stats.IncrementProcessed()
	}

	return writeErr
}

// processWrite formats and writes using handler-owned buffer under Lock.
// Used only by the single-consumer process() goroutine where contention
// is impossible, so the lock always succeeds immediately.
func (b *consoleBase) processWrite(entry *core.Entry, parBufPool *sync.Pool) error {
	if b.bufferFormatter != nil {
		b.mu.Lock()
		b.syncBuf.Reset()
		b.bufferFormatter.FormatEntry(entry, &b.syncBuf)
		_, err := b.writer.Write(b.syncBuf.Bytes())
		b.mu.Unlock()
		if err == nil {
			b.stats.IncrementProcessed()
		}
		return err
	}
	return b.write(entry, parBufPool)
}

// Stats returns a snapshot of the current statistics
func (b *consoleBase) Stats() handler.Snapshot {
	return b.stats.GetSnapshot()
}

// ConsoleConfig holds configuration for console handler
type ConsoleConfig struct {
	// Writer to write to (default: os.Stdout)
	Writer io.Writer
	// Formatter to use (default: TextFormatter)
	Formatter formatter.Formatter
	// Async enables asynchronous logging (default: true)
	Async bool
	// BufferSize is the size of the async queue (default: 1000)
	BufferSize int
	// OverflowPolicy defines per-level overflow behavior (default: uses DefaultLevelPolicy)
	OverflowPolicy map[core.Level]handler.OverflowPolicy
	// BlockTimeout is the timeout for blocking overflow policy (default: 100ms)
	BlockTimeout time.Duration
	// DrainTimeout is the timeout for draining queue on Close (default: 5s)
	DrainTimeout time.Duration
	// ConcurrentWriter indicates the Writer supports concurrent Write calls.
	// When true, the handler skips write-level locking for parallel log entries,
	// significantly improving parallel throughput. Automatically detected for
	// io.Discard and *os.File; set true for other goroutine-safe writers.
	ConcurrentWriter bool
}

// applyConsoleDefaults fills in zero-value fields with defaults.
func applyConsoleDefaults(cfg *ConsoleConfig) {
	if cfg.Writer == nil {
		cfg.Writer = os.Stdout
	}
	if cfg.Formatter == nil {
		cfg.Formatter = formatter.NewTextFormatter(formatter.Config{})
	}
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 1000
	}
	if cfg.OverflowPolicy == nil {
		cfg.OverflowPolicy = handler.DefaultLevelPolicy()
	}
	if cfg.BlockTimeout == 0 {
		cfg.BlockTimeout = 100 * time.Millisecond
	}
	if cfg.DrainTimeout == 0 {
		cfg.DrainTimeout = 5 * time.Second
	}
}

// NewConsoleHandler creates a new console handler.
// Returns a SyncConsoleHandler when Async is false, or an AsyncConsoleHandler
// when Async is true. Both implement Handler, FastHandler, and StatsProvider.
func NewConsoleHandler(cfg ConsoleConfig) handler.Handler {
	applyConsoleDefaults(&cfg)
	if cfg.Async {
		return newAsyncConsoleHandler(cfg)
	}
	return newSyncConsoleHandler(cfg)
}
