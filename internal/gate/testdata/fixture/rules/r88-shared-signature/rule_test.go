package r88

import "testing"

func TestAnswer(t *testing.T) {
	t.Log("checking the answer") // printed by both variants: it cannot tell them apart
	if got := Answer(); got != 42 {
		t.Fatalf("Answer() = %d, want 42", got)
	}
}
