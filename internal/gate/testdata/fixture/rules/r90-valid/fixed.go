//go:build !broken

package r90

import "sync"

// --8<-- [start:count]
func Count(workers, each int) int {
	var (
		mu sync.Mutex
		n  int
		wg sync.WaitGroup
	)
	for range workers {
		wg.Go(func() {
			for range each {
				mu.Lock()
				n++
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	return n
}

// --8<-- [end:count]
