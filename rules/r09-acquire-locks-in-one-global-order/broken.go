//go:build broken

package r09

// --8<-- [start:transfer]

// Transfer moves amount from one account to another. It locks from, then to:
// the order depends on the arguments, so Transfer(a, b) and Transfer(b, a)
// take the same two locks in opposite orders.
func Transfer(from, to *Account, amount int) {
	if from == to {
		return // one account: nothing moves, and a second Lock of a.mu would wait forever
	}
	from.mu.Lock()
	defer from.mu.Unlock()
	betweenLocks()
	to.mu.Lock()
	defer to.mu.Unlock()
	from.balance -= amount
	to.balance += amount
}

// --8<-- [end:transfer]
