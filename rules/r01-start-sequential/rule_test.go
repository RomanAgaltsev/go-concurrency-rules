package r01

import "testing"

func TestSumSquares(t *testing.T) {
	xs := make([]int, 1000)
	want := 0
	for i := range xs {
		xs[i] = i
		want += i * i
	}
	if got := SumSquares(xs); got != want {
		t.Fatalf("SumSquares = %d, want %d", got, want)
	}
}
