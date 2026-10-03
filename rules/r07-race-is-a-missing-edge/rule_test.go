package r07

import "testing"

// --8<-- [start:test]

func TestConcurrentInc(t *testing.T) {
	const workers, times = 4, 1000
	var c Counter
	IncConcurrently(&c, workers, times)
	if got, want := c.Value(), workers*times; got != want {
		t.Fatalf("lost %d of %d updates", want-got, want)
	}
}

// --8<-- [end:test]
