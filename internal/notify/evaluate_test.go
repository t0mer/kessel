package notify

import (
	"testing"

	"github.com/t0mer/kessel/internal/store"
)

func f64(v float64) *float64 { return &v }

func TestEvaluateAbsolute(t *testing.T) {
	rules := []store.ThresholdRule{{Category: CategoryPerformance, Mode: store.ThresholdAbsolute, Value: 80, Enabled: true}}
	run := store.Run{Perf: f64(70)}
	b := Evaluate(rules, run, nil)
	if len(b) != 1 {
		t.Fatalf("breaches = %d, want 1", len(b))
	}
	run.Perf = f64(85)
	if len(Evaluate(rules, run, nil)) != 0 {
		t.Fatal("85 >= 80 should not breach")
	}
}

func TestEvaluateDelta(t *testing.T) {
	rules := []store.ThresholdRule{{Category: CategorySEO, Mode: store.ThresholdDelta, Value: 10, Enabled: true}}
	run := store.Run{SEO: f64(70)}
	prev := store.Run{SEO: f64(90)} // dropped 20 > 10
	if len(Evaluate(rules, run, &prev)) != 1 {
		t.Fatal("drop of 20 should breach delta 10")
	}
	prev.SEO = f64(75) // dropped 5 < 10
	if len(Evaluate(rules, run, &prev)) != 0 {
		t.Fatal("drop of 5 should not breach delta 10")
	}
	if len(Evaluate(rules, run, nil)) != 0 {
		t.Fatal("delta with no previous run should not breach")
	}
}

func TestEvaluateSkipsNilScoreAndDisabled(t *testing.T) {
	rules := []store.ThresholdRule{
		{Category: CategoryPerformance, Mode: store.ThresholdAbsolute, Value: 80, Enabled: true}, // Perf nil -> skip
		{Category: CategorySEO, Mode: store.ThresholdAbsolute, Value: 80, Enabled: false},        // disabled
	}
	run := store.Run{SEO: f64(10)}
	if len(Evaluate(rules, run, nil)) != 0 {
		t.Fatal("nil score and disabled rule must be skipped")
	}
}
