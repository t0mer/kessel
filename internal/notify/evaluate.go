// Package notify evaluates thresholds and dispatches notifications.
package notify

import "github.com/t0mer/kessel/internal/store"

// Threshold categories.
const (
	CategoryPerformance   = "performance"
	CategoryAccessibility = "accessibility"
	CategoryBestPractices = "best_practices"
	CategorySEO           = "seo"
)

// Breach is a threshold rule that a run violated.
type Breach struct {
	Rule     store.ThresholdRule
	Current  *float64
	Previous *float64
}

func categoryScore(run store.Run, category string) *float64 {
	switch category {
	case CategoryPerformance:
		return run.Perf
	case CategoryAccessibility:
		return run.Accessibility
	case CategoryBestPractices:
		return run.BestPractices
	case CategorySEO:
		return run.SEO
	default:
		return nil
	}
}

// Evaluate returns the breaches for a run given its enabled rules and the
// previous successful run (for delta rules).
func Evaluate(rules []store.ThresholdRule, run store.Run, prev *store.Run) []Breach {
	var out []Breach
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		cur := categoryScore(run, rule.Category)
		if cur == nil {
			continue
		}
		switch rule.Mode {
		case store.ThresholdAbsolute:
			if *cur < rule.Value {
				out = append(out, Breach{Rule: rule, Current: cur})
			}
		case store.ThresholdDelta:
			if prev == nil {
				continue
			}
			p := categoryScore(*prev, rule.Category)
			if p == nil {
				continue
			}
			if (*p - *cur) > rule.Value {
				out = append(out, Breach{Rule: rule, Current: cur, Previous: p})
			}
		}
	}
	return out
}
