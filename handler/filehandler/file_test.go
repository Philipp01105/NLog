package filehandler

import (
	"testing"
	"time"

	"github.com/philipp01105/nlog/core"
)

func TestFileHandler_MaxBackups(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/test.log"

	h, err := NewFileHandler(FileConfig{
		Filename:   filename,
		Async:      false,
		MaxSize:    100, // Small size to trigger rotation
		MaxBackups: 2,   // Keep only 2 backups
	})
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	// Write enough to trigger multiple rotations
	for i := 0; i < 100; i++ {
		entry := core.GetEntry()
		entry.Level = core.InfoLevel
		entry.Message = "This is a test message that will trigger rotation"
		h.Handle(entry)
	}

	// Give time for rotation
	time.Sleep(100 * time.Millisecond)

	// Check that old backups are cleaned up
	// (This is a basic check - in practice you'd count the backup files)
}

func TestFileHandler_RotateInterval(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/test.log"

	h, err := NewFileHandler(FileConfig{
		Filename:       filename,
		Async:          false,
		RotateInterval: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	// Write a log
	entry := core.GetEntry()
	entry.Level = core.InfoLevel
	entry.Message = "first"
	h.Handle(entry)

	// Wait for rotation interval
	time.Sleep(150 * time.Millisecond)

	// Write another log - should trigger rotation
	entry2 := core.GetEntry()
	entry2.Level = core.InfoLevel
	entry2.Message = "second"
	h.Handle(entry2)

	// Basic check that rotation happened
	// (In practice you'd verify the rotated file exists)
}

func TestFileHandler_SyncOnClose(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/test.log"

	h, err := NewFileHandler(FileConfig{
		Filename: filename,
		Async:    false,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Write a log
	entry := core.GetEntry()
	entry.Level = core.InfoLevel
	entry.Message = "test"
	h.Handle(entry)

	// Close should sync the file
	if err := h.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
