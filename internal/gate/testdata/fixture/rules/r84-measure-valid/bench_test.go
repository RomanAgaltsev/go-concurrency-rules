package r84

import "testing"

func BenchmarkSum(b *testing.B) {
	xs := make([]int, 100)
	for b.Loop() {
		Sum(xs)
	}
}
