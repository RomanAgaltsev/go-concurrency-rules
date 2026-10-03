//go:build broken

package r89

import "time"

// Answer never returns in time: a deadlock with no watchdog looks like this to
// the gate. Each process must be stopped by its timeout, and the rule refused.
func Answer() int {
	time.Sleep(time.Hour)
	return 41
}
