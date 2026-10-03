//go:build !broken

package r77

// Sum adds xs in a loop.
func Sum(xs []int) int {
	total := 0
	for _, x := range xs {
		total += x
	}
	return total
}
