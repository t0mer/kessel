package runner

import (
	"testing"
	"time"
)

func TestSpacerReserveSpacesByMin(t *testing.T) {
	base := time.Unix(1000, 0)
	fake := base
	s := newSpacer(10*time.Second, func() time.Time { return fake })

	first := s.reserve()
	if !first.Equal(base) {
		t.Errorf("first reserve = %v, want %v", first, base)
	}
	second := s.reserve()
	if !second.Equal(base.Add(10 * time.Second)) {
		t.Errorf("second reserve = %v, want +10s", second)
	}
	third := s.reserve()
	if !third.Equal(base.Add(20 * time.Second)) {
		t.Errorf("third reserve = %v, want +20s", third)
	}
}

func TestSpacerZeroMinNoDelay(t *testing.T) {
	fake := time.Unix(1000, 0)
	s := newSpacer(0, func() time.Time { return fake })
	a := s.reserve()
	b := s.reserve()
	if !a.Equal(fake) || !b.Equal(fake) {
		t.Errorf("zero-min reserves should both be now: %v %v", a, b)
	}
}
