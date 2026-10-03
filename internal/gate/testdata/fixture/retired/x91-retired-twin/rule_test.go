package x91

import (
	"fmt"
	"testing"
)

func TestCapture(t *testing.T) {
	if got := fmt.Sprint(Capture()); got != "[0 1 2]" {
		t.Fatalf("closures saw %s, want [0 1 2]", got)
	}
}
