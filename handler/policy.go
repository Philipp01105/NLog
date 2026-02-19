package handler

import (
	"sync/atomic"

	"github.com/Philipp01105/NLog/core"
)

// OverflowPolicy defines how to handle full async queues
type OverflowPolicy int

const (
	// DropNewest drops the newest log entry when queue is full
	DropNewest OverflowPolicy = iota
	// DropOldest drops the oldest log entry when queue is full
	DropOldest
	// Block blocks the caller until space is available (with timeout)
	Block
)

// String returns the string representation of the policy
func (p OverflowPolicy) String() string {
	switch p {
	case DropNewest:
		return "DropNewest"
	case DropOldest:
		return "DropOldest"
	case Block:
		return "Block"
	default:
		return "Unknown"
	}
}

// DefaultLevelPolicy returns the default level-based overflow policies
func DefaultLevelPolicy() map[core.Level]OverflowPolicy {
	return map[core.Level]OverflowPolicy{
		core.DebugLevel: DropNewest, // Drop debug logs when full
		core.InfoLevel:  DropNewest, // Drop info logs when full
		core.WarnLevel:  DropNewest, // Drop warn logs when full
		core.ErrorLevel: Block,      // Block for errors (with timeout)
	}
}

// Stats tracks handler statistics
type Stats struct {
	// Separate atomic counters per level
	DroppedDebug uint64
	DroppedInfo  uint64
	DroppedWarn  uint64
	DroppedError uint64
	// BlockedTotal counts times logging blocked due to full queue
	BlockedTotal uint64
	// ProcessedTotal counts total processed logs
	ProcessedTotal uint64
}

// NewStats creates a new Stats instance
func NewStats() *Stats {
	return &Stats{}
}

// IncrementDropped atomically increments the dropped counter for a level
func (s *Stats) IncrementDropped(level core.Level) {
	switch level {
	case core.DebugLevel:
		atomic.AddUint64(&s.DroppedDebug, 1)
	case core.InfoLevel:
		atomic.AddUint64(&s.DroppedInfo, 1)
	case core.WarnLevel:
		atomic.AddUint64(&s.DroppedWarn, 1)
	case core.ErrorLevel:
		atomic.AddUint64(&s.DroppedError, 1)
	default:
		panic("unhandled default case, Please create a issue in github.com/Philipp01105/NLog")
	}
}

// IncrementBlocked atomically increments the blocked counter
func (s *Stats) IncrementBlocked() {
	atomic.AddUint64(&s.BlockedTotal, 1)
}

// IncrementProcessed atomically increments the processed counter
func (s *Stats) IncrementProcessed() {
	atomic.AddUint64(&s.ProcessedTotal, 1)
}

// GetDropped returns the dropped count for a level
func (s *Stats) GetDropped(level core.Level) uint64 {
	switch level {
	case core.DebugLevel:
		return atomic.LoadUint64(&s.DroppedDebug)
	case core.InfoLevel:
		return atomic.LoadUint64(&s.DroppedInfo)
	case core.WarnLevel:
		return atomic.LoadUint64(&s.DroppedWarn)
	case core.ErrorLevel:
		return atomic.LoadUint64(&s.DroppedError)
	default:
		return 0
	}
}

// GetBlocked returns the blocked count
func (s *Stats) GetBlocked() uint64 {
	return atomic.LoadUint64(&s.BlockedTotal)
}

// GetProcessed returns the processed count
func (s *Stats) GetProcessed() uint64 {
	return atomic.LoadUint64(&s.ProcessedTotal)
}

// GetTotalDropped returns the total dropped across all levels
func (s *Stats) GetTotalDropped() uint64 {
	return atomic.LoadUint64(&s.DroppedDebug) +
		atomic.LoadUint64(&s.DroppedInfo) +
		atomic.LoadUint64(&s.DroppedWarn) +
		atomic.LoadUint64(&s.DroppedError)
}

// Reset resets all counters to zero
func (s *Stats) Reset() {
	atomic.StoreUint64(&s.DroppedDebug, 0)
	atomic.StoreUint64(&s.DroppedInfo, 0)
	atomic.StoreUint64(&s.DroppedWarn, 0)
	atomic.StoreUint64(&s.DroppedError, 0)
	atomic.StoreUint64(&s.BlockedTotal, 0)
	atomic.StoreUint64(&s.ProcessedTotal, 0)
}

// Snapshot returns a snapshot of current stats
type Snapshot struct {
	DroppedTotal   map[core.Level]uint64
	BlockedTotal   uint64
	ProcessedTotal uint64
}

// GetSnapshot returns a snapshot of current statistics
func (s *Stats) GetSnapshot() Snapshot {
	return Snapshot{
		DroppedTotal: map[core.Level]uint64{
			core.DebugLevel: s.GetDropped(core.DebugLevel),
			core.InfoLevel:  s.GetDropped(core.InfoLevel),
			core.WarnLevel:  s.GetDropped(core.WarnLevel),
			core.ErrorLevel: s.GetDropped(core.ErrorLevel),
		},
		BlockedTotal:   s.GetBlocked(),
		ProcessedTotal: s.GetProcessed(),
	}
}
