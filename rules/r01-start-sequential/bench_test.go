package r01

import "testing"

// --8<-- [start:bench]

func BenchmarkSumSquares(b *testing.B) {
	xs := make([]int, 1000)
	for i := range xs {
		xs[i] = i
	}
	for b.Loop() {
		SumSquares(xs)
	}
}

// --8<-- [end:bench]
