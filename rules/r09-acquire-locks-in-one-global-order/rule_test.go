package r09

import (
	"sync"
	"testing"
	"time"
)

// --8<-- [start:test]

func TestOpposingTransfers(t *testing.T) {
	a := &Account{ID: 1, balance: 100}
	b := &Account{ID: 2, balance: 100}

	// Hold each Transfer after its first lock until both have one. With one
	// global order the second Transfer queues on the first lock and never
	// arrives, so the barrier gives up after 100ms and lets the first finish.
	var (
		mu      sync.Mutex
		arrived int
		both    = make(chan struct{})
	)
	betweenLocks = func() {
		mu.Lock()
		if arrived++; arrived == 2 {
			close(both)
		}
		mu.Unlock()
		select {
		case <-both:
		case <-time.After(100 * time.Millisecond):
		}
	}

	done := make(chan struct{})
	go func() {
		var wg sync.WaitGroup
		wg.Go(func() { Transfer(a, b, 10) })
		wg.Go(func() { Transfer(b, a, 20) })
		wg.Wait()
		close(done)
	}()

	// The watchdog: real time used to detect a failure, never to synchronise.
	// A deadlocked test would otherwise hang until -timeout.
	select {
	case <-done:
		// Reset only now: every goroutine that read the hook has finished,
		// and done orders their reads before this write.
		betweenLocks = func() {}
	case <-time.After(2 * time.Second):
		t.Fatal("watchdog: deadlock — Transfer(a, b) and Transfer(b, a) each hold the lock the other needs")
	}
	if a.Balance() != 110 || b.Balance() != 90 {
		t.Fatalf("balances %d and %d, want 110 and 90", a.Balance(), b.Balance())
	}
}

// --8<-- [end:test]
