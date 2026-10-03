//go:build !broken

package r12

import "sync"

// --8<-- [start:stats]

// Stats counts hits from many goroutines. It must not be copied after first
// use, so every method takes a pointer.
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

// Hits returns the count, under the same mutex Hit locks.
func (s *Stats) Hits() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hits
}

// --8<-- [end:stats]
