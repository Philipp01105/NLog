// Package core defines the shared types used across the NLog framework.
//
// It provides the Level type for severity filtering, the Entry type that
// represents a single log event, and the Field type for zero-allocation
// structured key-value pairs.
//
// Entry objects are pooled via sync.Pool to keep the hot path
// allocation-free. Callers get an Entry with GetEntry and must
// return it with PutEntry once the handler has consumed it. The pool
// pre-allocates the Fields slice with capacity 8, which covers most
// log calls without triggering a slice growth.
//
// Field encodes values into fixed-size numeric fields (Int64, Float64)
// wherever possible so that common types like int, bool, and time.Time
// never escape to the heap. The Any field exists as a fallback for
// arbitrary types but will cause an allocation.
package core
