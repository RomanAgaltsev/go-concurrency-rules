//go:build !broken

package r80

import "sync"

type Stats struct {
	mu   sync.Mutex
	hits int
}

func (s *Stats) Hit() { s.mu.Lock(); s.hits++; s.mu.Unlock() }

func (s *Stats) Hits() int { s.mu.Lock(); defer s.mu.Unlock(); return s.hits }
