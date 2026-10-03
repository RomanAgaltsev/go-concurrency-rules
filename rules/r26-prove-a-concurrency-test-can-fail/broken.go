//go:build broken

package r26

import "testing"

// --8<-- [start:check]

// CheckRunsOnce fails t if newIniter's Initer runs f more than once.
//
// It calls Do twice, one after the other — so the second call always sees the
// first one finished, and no implementation can fail it. It has never been
// seen to fail, and it cannot.
func CheckRunsOnce(t testing.TB, newIniter func() Initer) {
	t.Helper()
	o := newIniter()
	calls := 0
	o.Do(func() { calls++ })
	o.Do(func() { calls++ })
	if calls != 1 {
		t.Errorf("f ran %d times, want 1", calls)
	}
}

// --8<-- [end:check]
