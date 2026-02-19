package handler

import (
	"bufio"
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
)

// FileHandler writes log entries to a file with rotation support
type FileHandler struct {
	filename        string
	file            *os.File
	bufWriter       *bufio.Writer
	sizeWriter      *sizeTrackingWriter
	formatter       formatter.Formatter
	writerFormatter formatter.WriterFormatter
	async           bool
	queue           chan *core.Entry
	wg              sync.WaitGroup
	closed          chan struct{}
	mu              sync.Mutex
	maxSize         int64
	maxAge          time.Duration
	maxBackups      int
	rotateInterval  time.Duration
	currentSize     int64
	lastRotateTime  time.Time
	overflowPolicy  map[core.Level]OverflowPolicy
	blockTimeout    time.Duration
	stats           *Stats
	drainTimeout    time.Duration
	blockTimer      *time.Timer
}

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
	OverflowPolicy map[core.Level]OverflowPolicy
	// BlockTimeout is the timeout for blocking overflow policy (default: 100ms)
	BlockTimeout time.Duration
	// DrainTimeout is the timeout for draining queue on Close (default: 5s)
	DrainTimeout time.Duration
}

// NewFileHandler creates a new file handler
func NewFileHandler(cfg FileConfig) (*FileHandler, error) {
	if cfg.Filename == "" {
		return nil, fmt.Errorf("filename is required")
	}
	if cfg.Formatter == nil {
		cfg.Formatter = formatter.NewTextFormatter(formatter.Config{})
	}
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 1000
	}
	if cfg.OverflowPolicy == nil {
		cfg.OverflowPolicy = DefaultLevelPolicy()
	}
	if cfg.BlockTimeout == 0 {
		cfg.BlockTimeout = 100 * time.Millisecond
	}
	if cfg.DrainTimeout == 0 {
		cfg.DrainTimeout = 5 * time.Second
	}

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
		err := file.Close()
		if err != nil {
			return nil, err
		}
		return nil, err
	}

	sw := &sizeTrackingWriter{w: file}
	h := &FileHandler{
		filename:       cfg.Filename,
		file:           file,
		sizeWriter:     sw,
		bufWriter:      bufio.NewWriterSize(sw, 4096),
		formatter:      cfg.Formatter,
		async:          cfg.Async,
		maxSize:        cfg.MaxSize,
		maxAge:         cfg.MaxAge,
		maxBackups:     cfg.MaxBackups,
		rotateInterval: cfg.RotateInterval,
		currentSize:    info.Size(),
		lastRotateTime: time.Now(),
		closed:         make(chan struct{}),
		overflowPolicy: cfg.OverflowPolicy,
		blockTimeout:   cfg.BlockTimeout,
		stats:          NewStats(),
		drainTimeout:   cfg.DrainTimeout,
		blockTimer:     newStoppedTimer(),
	}

	// Cache WriterFormatter for zero-alloc path
	h.writerFormatter, _ = cfg.Formatter.(formatter.WriterFormatter)

	if h.async {
		h.queue = make(chan *core.Entry, cfg.BufferSize)
		h.wg.Add(1)
		go h.process()
	}

	return h, nil
}

// Handle processes a log entry
func (h *FileHandler) Handle(entry *core.Entry) error {
	if !h.async {
		return h.write(entry)
	}

	// Get overflow policy for this level
	policy, ok := h.overflowPolicy[entry.Level]
	if !ok {
		policy = DropNewest // Default if not specified
	}

	switch policy {
	case Block:
		// Try to send with timeout using reusable timer
		select {
		case h.queue <- entry:
			return nil
		default:
			// Queue full, use timer for timeout
			if !h.blockTimer.Stop() {
				select {
				case <-h.blockTimer.C:
				default:
				}
			}
			h.blockTimer.Reset(h.blockTimeout)
			select {
			case h.queue <- entry:
				if !h.blockTimer.Stop() {
					select {
					case <-h.blockTimer.C:
					default:
					}
				}
				return nil
			case <-h.blockTimer.C:
				// Timeout - fall back to synchronous write
				h.stats.IncrementBlocked()
				return h.write(entry)
			case <-h.closed:
				// Handler is closing, write synchronously
				if !h.blockTimer.Stop() {
					select {
					case <-h.blockTimer.C:
					default:
					}
				}
				return h.write(entry)
			}
		}

	case DropOldest:
		// Try non-blocking send
		select {
		case h.queue <- entry:
			return nil
		default:
			// Queue full - try to drop oldest
			select {
			case <-h.queue: // Remove oldest
				h.stats.IncrementDropped(entry.Level)
			default:
			}
			// Try again
			select {
			case h.queue <- entry:
				return nil
			default:
				// Still full, drop this one
				h.stats.IncrementDropped(entry.Level)
				return nil
			}
		}

	case DropNewest:
		fallthrough
	default:
		// Non-blocking send
		select {
		case h.queue <- entry:
			return nil
		default:
			// Queue full - drop this entry
			h.stats.IncrementDropped(entry.Level)
			return nil
		}
	}
}

// write formats and writes an entry
func (h *FileHandler) write(entry *core.Entry) error {
	if h.writerFormatter != nil {
		h.mu.Lock()
		defer h.mu.Unlock()

		if err := h.rotateIfNeeded(); err != nil {
			return err
		}

		prevFlushed := h.sizeWriter.written
		prevBuffered := h.bufWriter.Buffered()
		err := h.writerFormatter.FormatTo(entry, h.bufWriter)
		if err == nil {
			written := (h.sizeWriter.written - prevFlushed) + int64(h.bufWriter.Buffered()-prevBuffered)
			h.currentSize += written
			h.stats.IncrementProcessed()
		}
		return err
	}

	data, err := h.formatter.Format(entry)
	if err != nil {
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Check if rotation is needed
	if err := h.rotateIfNeeded(); err != nil {
		return err
	}

	n, err := h.bufWriter.Write(data)
	if err == nil {
		h.currentSize += int64(n)
		h.stats.IncrementProcessed()
	}

	return err
}

// CanRecycleEntry returns true if the caller can recycle the entry after Handle returns
func (h *FileHandler) CanRecycleEntry() bool {
	return !h.async
}

// rotateIfNeeded checks and performs rotation if needed
func (h *FileHandler) rotateIfNeeded() error {
	needRotate := false

	// Check size-based rotation
	if h.maxSize > 0 && h.currentSize >= h.maxSize {
		needRotate = true
	}

	// Check time-based rotation (by age)
	if h.maxAge > 0 && time.Since(h.lastRotateTime) >= h.maxAge {
		needRotate = true
	}

	// Check interval-based rotation
	if h.rotateInterval > 0 && time.Since(h.lastRotateTime) >= h.rotateInterval {
		needRotate = true
	}

	if !needRotate {
		return nil
	}

	return h.rotate()
}

// rotate performs the actual file rotation
func (h *FileHandler) rotate() error {
	// Flush buffered writer, sync and close current file
	if err := h.bufWriter.Flush(); err != nil {
		return err
	}
	if err := h.file.Sync(); err != nil {
		return err
	}
	if err := h.file.Close(); err != nil {
		return err
	}

	// Rename current file with timestamp
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	rotatedName := fmt.Sprintf("%s.%s", h.filename, timestamp)

	if err := os.Rename(h.filename, rotatedName); err != nil {
		// If rename fails, try to reopen the original file
		file, openErr := os.OpenFile(h.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if openErr != nil {
			return fmt.Errorf("rotation failed: %v, reopen failed: %v", err, openErr)
		}
		h.file = file
		return err
	}

	// Clean up old backups if needed
	if h.maxBackups > 0 {
		h.cleanupOldBackups()
	}

	// Open new file
	file, err := os.OpenFile(h.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	h.file = file
	h.sizeWriter.reset(file)
	h.bufWriter.Reset(h.sizeWriter)
	h.currentSize = 0
	h.lastRotateTime = time.Now()

	return nil
}

// cleanupOldBackups removes old backup files based on MaxBackups
func (h *FileHandler) cleanupOldBackups() {
	dir := filepath.Dir(h.filename)
	base := filepath.Base(h.filename)

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
	if len(backups) > h.maxBackups {
		toRemove := backups[:len(backups)-h.maxBackups]
		for _, file := range toRemove {
			err := os.Remove(file)
			if err != nil {
				return
			}
		}
	}
}

// process handles async log processing
func (h *FileHandler) process() {
	defer h.wg.Done()

	for {
		select {
		case entry := <-h.queue:
			err := h.write(entry)
			if err != nil {
				return
			}
			core.PutEntry(entry)
		case <-h.closed:
			// Drain remaining entries with timeout
			deadline := time.After(h.drainTimeout)
		drainLoop:
			for {
				select {
				case entry := <-h.queue:
					err := h.write(entry)
					if err != nil {
						return
					}
					core.PutEntry(entry)
				case <-deadline:
					// Timeout reached, stop draining
					break drainLoop
				default:
					// Queue empty
					break drainLoop
				}
			}
			return
		}
	}
}

// Stats returns a snapshot of the current statistics
func (h *FileHandler) Stats() Snapshot {
	return h.stats.GetSnapshot()
}

// Close closes the handler
func (h *FileHandler) Close() error {
	// Check if already closed (without lock to avoid deadlock)
	select {
	case <-h.closed:
		return nil // Already closed
	default:
	}

	if h.async {
		close(h.closed)
		h.wg.Wait() // Wait without holding lock to avoid deadlock

		h.mu.Lock()
		close(h.queue)
		h.mu.Unlock()
	}

	// Sync and close file
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.file != nil {
		flushErr := h.bufWriter.Flush()
		if flushErr != nil {
			h.file.Close()
			return flushErr
		}
		syncErr := h.file.Sync()
		if syncErr != nil {
			h.file.Close()
			return syncErr
		}
		return h.file.Close()
	}

	return nil
}
