//go:build !broken

package r01

// --8<-- [start:sum]

// SumSquares returns the sum of the squares of xs, in a loop.
func SumSquares(xs []int) int {
	total := 0
	for _, x := range xs {
		total += x * x
	}
	return total
}

// --8<-- [end:sum]
