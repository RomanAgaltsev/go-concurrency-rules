//go:build broken

package r07

import "sync"

// --8<-- [start:counter]

// Counter counts events reported by many goroutines.
type Counter struct {
	mu sync.Mutex // guards n
	n  int
}

// Inc adds one. Each Unlock happens before the next Lock of the same mutex,
// so every Inc is ordered before or after every other: the missing edge.
func (c *Counter) Inc() {
	c.mu.Lock()
	c.n++
	c.mu.Unlock()
}

// Value returns the count.
func (c *Counter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

// --8<-- [end:counter]
