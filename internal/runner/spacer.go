package runner

import (
	"context"
	"sync"
	"time"
)

// spacer enforces a minimum gap between successive reservations, so PSI API
// calls are spaced out regardless of how many goroutines request them.
type spacer struct {
	mu   sync.Mutex
	min  time.Duration
	next time.Time
	now  func() time.Time
}

func newSpacer(min time.Duration, now func() time.Time) *spacer {
	return &spacer{min: min, now: now}
}

// reserve returns the earliest time the caller may proceed and advances the
// internal clock so the next reservation is at least min later.
func (s *spacer) reserve() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	at := now
	if s.next.After(now) {
		at = s.next
	}
	s.next = at.Add(s.min)
	return at
}

// wait blocks until this caller's reserved time, honoring ctx cancellation.
func (s *spacer) wait(ctx context.Context) error {
	if s.min <= 0 {
		return nil
	}
	at := s.reserve()
	d := at.Sub(s.now())
	if d <= 0 {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
