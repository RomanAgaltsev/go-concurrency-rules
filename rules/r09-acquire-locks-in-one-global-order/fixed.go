//go:build !broken

package r09

// --8<-- [start:transfer]

// Transfer moves amount from one account to another. It locks the account
// with the lower ID first, whatever the direction of the transfer: every
// goroutine takes any two locks in the same global order, so none can hold
// the lock another one needs while waiting for a lock that one holds.
func Transfer(from, to *Account, amount int) {
	if from == to {
		return // one account: nothing moves, and a second Lock of a.mu would wait forever
	}
	first, second := from, to
	if second.ID < first.ID {
		first, second = second, first
	}
	first.mu.Lock()
	defer first.mu.Unlock()
	betweenLocks()
	second.mu.Lock()
	defer second.mu.Unlock()
	from.balance -= amount
	to.balance += amount
}

// --8<-- [end:transfer]
