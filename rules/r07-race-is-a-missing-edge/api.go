// Package r07 is rule R07: a data race is a missing happens-before edge.
//
// broken.go and fixed.go both define Counter; the build tag "broken" selects
// which one is compiled, and rule_test.go is the proof against either.
package r07

import "sync"

// --8<-- [start:drive]

// IncConcurrently calls c.Inc times times from each of workers goroutines and
// waits for them all. Wait gives the caller an edge from every Inc to its next
// line; it orders nothing between one Inc and another.
func IncConcurrently(c *Counter, workers, times int) {
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for range times {
				c.Inc()
			}
		})
	}
	wg.Wait()
}

// --8<-- [end:drive]
