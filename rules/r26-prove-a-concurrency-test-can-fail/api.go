// Package r26 is rule R26: prove a concurrency test can fail.
//
// Here the unit under test is a check: CheckRunsOnce, which should fail on any
// Initer that can run its function twice. broken.go and fixed.go each define
// a version of it; rule_test.go runs it against a known-buggy Initer and
// requires it to fail there.
package r26

import "sync/atomic"

// --8<-- [start:initer]

// Initer runs a function at most once, however many goroutines call Do.
// *sync.Once is one.
type Initer interface {
	Do(f func())
}

// FlakyOnce is a known-buggy Initer. Its flag is atomic, so the race detector
// has nothing to report; its bug is check-then-act: two callers that both see
// done == false both run f.
type FlakyOnce struct {
	done atomic.Bool
}

// Do runs f unless a call has already finished.
func (o *FlakyOnce) Do(f func()) {
	if o.done.Load() {
		return
	}
	f()
	o.done.Store(true)
}

// --8<-- [end:initer]
