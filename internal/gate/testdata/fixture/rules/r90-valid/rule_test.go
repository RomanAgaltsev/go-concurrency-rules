package r90

import "testing"

func TestCount(t *testing.T) {
	if got := Count(4, 1000); got != 4000 {
		t.Fatalf("lost updates: got %d, want 4000", got)
	}
}
