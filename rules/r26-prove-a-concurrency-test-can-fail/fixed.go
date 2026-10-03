//go:build !broken

package r26

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// --8<-- [start:check]

// CheckRunsOnce fails t if newIniter's Initer runs f more than once.
//
// It makes two goroutines call Do at the same moment and holds the first call
// to f open until the second goroutine has started: an Initer that lets both
// in is caught every time, not once in a million runs. A correct Initer keeps
// the second caller waiting inside Do; the first call then gives up waiting
// after 100ms and returns — time bounds the wait, it orders nothing.
func CheckRunsOnce(t testing.TB, newIniter func() Initer) {
	t.Helper()
	o := newIniter()
	var calls atomic.Int32
	secondStarted := make(chan struct{})
	secondInF := make(chan struct{})
	f := func() {
		if calls.Add(1) == 2 {
			close(secondInF)
			return
		}
		<-secondStarted
		select {
		case <-secondInF:
		case <-time.After(100 * time.Millisecond):
		}
	}
	var wg sync.WaitGroup
	wg.Go(func() { o.Do(f) })
	wg.Go(func() {
		close(secondStarted)
		o.Do(f)
	})
	wg.Wait()
	if n := calls.Load(); n != 1 {
		t.Errorf("f ran %d times, want 1", n)
	}
}

// --8<-- [end:check]
