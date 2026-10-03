//go:build !broken

package r87

import "time"

// Answer hangs: a fixed variant that deadlocks. The fixed run must be stopped
// by its timeout and the message must say so.
func Answer() int {
	time.Sleep(time.Hour)
	return 42
}
