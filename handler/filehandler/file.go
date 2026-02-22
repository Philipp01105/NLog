package filehandler

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/philipp01105/nlog/core"
	"github.com/philipp01105/nlog/formatter"
	"github.com/philipp01105/nlog/handler"
)

// sizeTrackingWriter wraps an io.Writer and tracks total bytes written
type sizeTrackingWriter struct {
	w       io.Writer
	written int64
}

func (s *sizeTrackingWriter) Write(p []byte) (n int, err error) {
	n, err = s.w.Write(p)
	s.written += int64(n)
	return
}

func (s *sizeTrackingWriter) reset(w io.Writer) {
	s.w = w
	s.written = 0
}

// fileBase contains shared fields and methods for file handlers.
type fileBase struct {
	filename        string
	file            *os.File
	bufWriter       *bufio.Writer
	sizeWriter      *sizeTrackingWriter
	formatter       formatter.Formatter
	writerFormatter formatter.WriterFormatter
	bufferFormatter formatter.BufferFormatter
	mu              sync.Mutex
	syncBuf         bytes.Buffer
	maxSize         int64
	maxAge          time.Duration
	maxBackups      int
	rotateInterval  time.Duration
	currentSize     int64
	lastRotateTime  time.Time
	hasRotation     bool
	stats           *handler.Stats
	closed          chan struct{}
}

// write formats and writes an entry
func (b *fileBase) write(entry *core.Entry) error {
	// BufferFormatter fast path: format into handler-owned buffer, write to bufio.Writer.
	// Avoids buffer pool get/put overhead.
	if b.bufferFormatter != nil {
		b.mu.Lock()
		if err := b.rotateIfNeeded(); err != nil {
			b.mu.Unlock()
			return err
		}

		b.syncBuf.Reset()
		b.bufferFormatter.FormatEntry(entry, &b.syncBuf)
		n, err := b.bufWriter.Write(b.syncBuf.Bytes())
		if err == nil {
			b.currentSize += int64(n)
			b.stats.IncrementProcessed()
		}
		b.mu.Unlock()
		return err
	}

	if b.writerFormatter != nil {
		b.mu.Lock()
		if err := b.rotateIfNeeded(); err != nil {
			b.mu.Unlock()
			return err
		}

		prevFlushed := b.sizeWriter.written
		prevBuffered := b.bufWriter.Buffered()
		err := b.writerFormatter.FormatTo(entry, b.bufWriter)
		if err == nil {
			written := (b.sizeWriter.written - prevFlushed) + int64(b.bufWriter.Buffered()-prevBuffered)
			b.currentSize += written
			b.stats.IncrementProcessed()
		}
		b.mu.Unlock()
		return err
	}

	data, err := b.formatter.Format(entry)
	if err != nil {
		return err
	}

	b.mu.Lock()
	if err := b.rotateIfNeeded(); err != nil {
		b.mu.Unlock()
		return err
	}

	n, err := b.bufWriter.Write(data)
	if err == nil {
		b.currentSize += int64(n)
		b.stats.IncrementProcessed()
	}
	b.mu.Unlock()

	return err
}

// rotateIfNeeded checks and performs rotation if needed
func (b *fileBase) rotateIfNeeded() error {
	if !b.hasRotation {
		return nil
	}

	needRotate := false

	// Check size-based rotation
	if b.maxSize > 0 && b.currentSize >= b.maxSize {
		needRotate = true
	}

	// Check time-based rotation (by age)
	if b.maxAge > 0 && time.Since(b.lastRotateTime) >= b.maxAge {
		needRotate = true
	}

	// Check interval-based rotation
	if b.rotateInterval > 0 && time.Since(b.lastRotateTime) >= b.rotateInterval {
		needRotate = true
	}

	if !needRotate {
		return nil
	}

	return b.rotate()
}

// rotate performs the actual file rotation
func (b *fileBase) rotate() error {
	// Flush buffered writer, sync and close current file
	if err := b.bufWriter.Flush(); err != nil {
		return err
	}
	if err := b.file.Sync(); err != nil {
		return err
	}
	if err := b.file.Close(); err != nil {
		return err
	}

	// Rename current file with timestamp
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	rotatedName := fmt.Sprintf("%s.%s", b.filename, timestamp)

	if err := os.Rename(b.filename, rotatedName); err != nil {
		// If rename fails, try to reopen the original file
		file, openErr := os.OpenFile(b.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if openErr != nil {
			return fmt.Errorf("rotation failed: %v, reopen failed: %v", err, openErr)
		}
		b.file = file
		return err
	}

	// Clean up old backups if needed
	if b.maxBackups > 0 {
		b.cleanupOldBackups()
	}

	// Open new file
	file, err := os.OpenFile(b.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	b.file = file
	b.sizeWriter.reset(file)
	b.bufWriter.Reset(b.sizeWriter)
	b.currentSize = 0
	b.lastRotateTime = time.Now()

	return nil
}

// cleanupOldBackups removes old backup files based on MaxBackups
func (b *fileBase) cleanupOldBackups() {
	dir := filepath.Dir(b.filename)
	base := filepath.Base(b.filename)

	// Find all backup files
	pattern := filepath.Join(dir, base+".*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}

	// Filter to only timestamp-based backups
	var backups []string
	for _, match := range matches {
		if strings.HasPrefix(filepath.Base(match), base+".") {
			backups = append(backups, match)
		}
	}

	// Sort by modification time (oldest first)
	sort.Slice(backups, func(i, j int) bool {
		infoI, errI := os.Stat(backups[i])
		infoJ, errJ := os.Stat(backups[j])
		if errI != nil || errJ != nil {
			return false
		}
		return infoI.ModTime().Before(infoJ.ModTime())
	})

	// Remove oldest files if we exceed MaxBackups
	if len(backups) > b.maxBackups {
		toRemove := backups[:len(backups)-b.maxBackups]
		for _, file := range toRemove {
			err := os.Remove(file)
			if err != nil {
				return
			}
		}
	}
}

// Stats returns a snapshot of the current statistics
func (b *fileBase) Stats() handler.Snapshot {
	return b.stats.GetSnapshot()
}

// closeFile flushes, syncs and closes the underlying file.
func (b *fileBase) closeFile() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.file != nil {
		flushErr := b.bufWriter.Flush()
		if flushErr != nil {
			b.file.Close()
			return flushErr
		}
		syncErr := b.file.Sync()
		if syncErr != nil {
			b.file.Close()
			return syncErr
		}
		return b.file.Close()
	}

	return nil
}

// FileConfig holds configuration for file handler
type FileConfig struct {
	// Filename is the path to the log file
	Filename string
	// Formatter to use (default: TextFormatter)
	Formatter formatter.Formatter
	// Async enables asynchronous logging (default: true)
	Async bool
	// BufferSize is the size of the async queue (default: 1000)
	BufferSize int
	// MaxSize is the maximum size in bytes before rotation (0 = no size rotation)
	MaxSize int64
	// MaxAge is the maximum age before rotation (0 = no time rotation)
	MaxAge time.Duration
	// MaxBackups is the maximum number of old log files to retain (0 = keep all)
	MaxBackups int
	// RotateInterval is the interval for time-based rotation (0 = no interval rotation)
	RotateInterval time.Duration
	// OverflowPolicy defines per-level overflow behavior (default: uses DefaultLevelPolicy)
	OverflowPolicy map[core.Level]handler.OverflowPolicy
	// BlockTimeout is the timeout for blocking overflow policy (default: 100ms)
	BlockTimeout time.Duration
	// DrainTimeout is the timeout for draining queue on Close (default: 5s)
	DrainTimeout time.Duration
}

// applyFileDefaults fills in zero-value fields with defaults.
func applyFileDefaults(cfg *FileConfig) {
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

// initFileBase initializes a fileBase in place with the given config and opened file.
func initFileBase(b *fileBase, cfg FileConfig, file *os.File, fileSize int64) {
	sw := &sizeTrackingWriter{w: file}
	b.filename = cfg.Filename
	b.file = file
	b.sizeWriter = sw
	b.bufWriter = bufio.NewWriterSize(sw, 4096)
	b.formatter = cfg.Formatter
	b.maxSize = cfg.MaxSize
	b.maxAge = cfg.MaxAge
	b.maxBackups = cfg.MaxBackups
	b.rotateInterval = cfg.RotateInterval
	b.currentSize = fileSize
	b.lastRotateTime = time.Now()
	b.hasRotation = cfg.MaxSize > 0 || cfg.MaxAge > 0 || cfg.RotateInterval > 0
	b.closed = make(chan struct{})
	b.stats = handler.NewStats()

	// Cache WriterFormatter for zero-alloc path
	b.writerFormatter, _ = cfg.Formatter.(formatter.WriterFormatter)

	// Cache BufferFormatter for sync fast path (avoids buffer pool + direct bufio write)
	b.bufferFormatter, _ = cfg.Formatter.(formatter.BufferFormatter)

	// Pre-grow sync buffer for handler-owned format path
	if b.bufferFormatter != nil {
		b.syncBuf.Grow(256)
	}
}

// NewFileHandler creates a new file handler.
// Returns a SyncFileHandler when Async is false, or an AsyncFileHandler
// when Async is true. Both implement Handler, FastHandler, and StatsProvider.
func NewFileHandler(cfg FileConfig) (handler.Handler, error) {
	if cfg.Filename == "" {
		return nil, fmt.Errorf("filename is required")
	}
	applyFileDefaults(&cfg)

	// Create directory if it doesn't exist
	dir := filepath.Dir(cfg.Filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Open file
	file, err := os.OpenFile(cfg.Filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	// Get file size
	info, err := file.Stat()
	if err != nil {
		closeErr := file.Close()
		if closeErr != nil {
			return nil, closeErr
		}
		return nil, err
	}

	if cfg.Async {
		return newAsyncFileHandler(cfg, file, info.Size()), nil
	}
	return newSyncFileHandler(cfg, file, info.Size()), nil
}
