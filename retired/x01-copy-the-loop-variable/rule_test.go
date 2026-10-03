package x01

import (
	"fmt"
	"testing"
)

// --8<-- [start:test]

func TestCapture(t *testing.T) {
	if got := fmt.Sprint(Capture()); got != "[0 1 2]" {
		t.Fatalf("closures saw %s, want [0 1 2]", got)
	}
}

// --8<-- [end:test]
