//go:build broken

package r01

import "sync"

// --8<-- [start:sum]

// SumSquares returns the sum of the squares of xs, one goroutine per element.
// Each goroutine writes only its own slot, so there is nothing to lock — and
// still every element pays for a goroutine, a closure and a WaitGroup update
// to save one multiplication.
func SumSquares(xs []int) int {
	sq := make([]int, len(xs))
	var wg sync.WaitGroup
	for i, x := range xs {
		wg.Go(func() { sq[i] = x * x })
	}
	wg.Wait()
	total := 0
	for _, s := range sq {
		total += s
	}
	return total
}

// --8<-- [end:sum]
