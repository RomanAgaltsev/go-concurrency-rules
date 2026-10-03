//go:build broken

package r12

import "sync"

// --8<-- [start:stats]

// Stats counts hits from many goroutines.
type Stats struct {
	mu   sync.Mutex // guards hits
	hits int
}

// Hit records one hit.
func (s *Stats) Hit() {
	s.mu.Lock()
	s.hits++
	s.mu.Unlock()
}

// Hits returns the count. Its value receiver copies the whole Stats — the
// mutex included — so it locks a copy no other goroutine has ever seen, and
// reads hits while Hit is writing it.
func (s Stats) Hits() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hits
}

// --8<-- [end:stats]
