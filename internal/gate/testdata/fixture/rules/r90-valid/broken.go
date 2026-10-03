//go:build broken

package r90

import "sync"

// --8<-- [start:count]
func Count(workers, each int) int {
	var n int
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for range each {
				n++
			}
		})
	}
	wg.Wait()
	return n
}

// --8<-- [end:count]
