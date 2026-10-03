// Package r09 is rule R09: acquire locks in one global order.
//
// broken.go and fixed.go both define Transfer; the build tag "broken" selects
// which one is compiled, and rule_test.go is the proof against either.
package r09

import "sync"

// --8<-- [start:account]

// Account is a balance guarded by its own mutex.
type Account struct {
	ID      int
	mu      sync.Mutex // guards balance
	balance int
}

// Balance returns the current balance.
func (a *Account) Balance() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.balance
}

// --8<-- [end:account]

// betweenLocks runs after Transfer takes its first lock and before it asks for
// the second. It does nothing in a program; the proof uses it to make both
// goroutines hold their first lock at once — the interleaving that deadlocks,
// on demand instead of once in a million runs.
var betweenLocks = func() {}
