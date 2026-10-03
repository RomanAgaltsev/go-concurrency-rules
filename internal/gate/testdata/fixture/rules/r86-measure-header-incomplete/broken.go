//go:build broken

package r86

import "sync"

// Sum adds xs with one goroutine per element.
func Sum(xs []int) int {
	var (
		mu    sync.Mutex
		total int
		wg    sync.WaitGroup
	)
	for _, x := range xs {
		wg.Go(func() {
			mu.Lock()
			total += x
			mu.Unlock()
		})
	}
	wg.Wait()
	return total
}
