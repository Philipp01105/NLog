package core

import (
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

var (
	coarseClockOnce sync.Once
	coarseNow       unsafe.Pointer // *time.Time
)

// StartCoarseClock starts the background goroutine that caches
// time.Now() every 500µs. It is safe to call multiple times; the
// goroutine is started exactly once. The goroutine runs for the
// lifetime of the process; this is intentional because logging
// typically spans the entire application lifecycle.
func StartCoarseClock() {
	coarseClockOnce.Do(func() {
		t := time.Now()
		atomic.StorePointer(&coarseNow, unsafe.Pointer(&t))
		go func() {
			ticker := time.NewTicker(500 * time.Microsecond)
			for range ticker.C {
				t := time.Now()
				atomic.StorePointer(&coarseNow, unsafe.Pointer(&t))
			}
		}()
	})
}

// CoarseNow returns the most recently cached time.Time value.
// StartCoarseClock must have been called before using CoarseNow.
func CoarseNow() time.Time {
	return *(*time.Time)(atomic.LoadPointer(&coarseNow))
}
