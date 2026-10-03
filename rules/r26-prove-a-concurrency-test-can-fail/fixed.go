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
// Two goroutines each announce themselves and then call Do. Whichever call
// reaches f first holds it open until both goroutines have announced — so the
// other one is, at most, a few instructions away from calling Do — and then
// waits up to 100ms for that call to arrive in f. An Initer that lets both in
// is caught every time, not once in a million runs. A correct Initer keeps the
// second caller waiting inside Do, so the first call gives up after 100ms and
// returns — time bounds the wait, it orders nothing.
func CheckRunsOnce(t testing.TB, newIniter func() Initer) {
	t.Helper()
	o := newIniter()
	var calls, started atomic.Int32
	bothStarted := make(chan struct{})
	secondInF := make(chan struct{})
	f := func() {
		if calls.Add(1) == 2 {
			close(secondInF)
			return
		}
		<-bothStarted
		select {
		case <-secondInF:
		case <-time.After(100 * time.Millisecond):
		}
	}
	call := func() {
		if started.Add(1) == 2 {
			close(bothStarted)
		}
		o.Do(f)
	}
	var wg sync.WaitGroup
	wg.Go(call)
	wg.Go(call)
	wg.Wait()
	if n := calls.Load(); n != 1 {
		t.Errorf("f ran %d times, want 1", n)
	}
}

// --8<-- [end:check]
