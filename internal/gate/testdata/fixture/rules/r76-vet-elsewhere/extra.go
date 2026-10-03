//go:build broken

package r76

import "sync"

// Elsewhere copies a lock — in a file that is not broken.go. broken.go itself
// is identical to fixed.go, so the rule's broken variant proves nothing.
func Elsewhere(mu sync.Mutex) { _ = mu }
