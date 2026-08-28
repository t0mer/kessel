// Package psi is a client for the Google PageSpeed Insights v5 API.
package psi

import (
	"encoding/json"
	"fmt"
	"math"
)

// Result is the extracted, scalar view of a PSI run. Score pointers are nil
// when the API omitted that category; lab-metric pointers are nil when the
// corresponding audit was absent.
type Result struct {
	Performance   *float64 // 0-100
	Accessibility *float64
	BestPractices *float64
	SEO           *float64
	Lab           LabMetrics
	Field         *FieldData // CrUX field data; nil when not present
}

// LabMetrics holds Core Web Vitals lab (Lighthouse) metrics.
type LabMetrics struct {
	LCPms *float64
	CLS   *float64
	TBTms *float64
	FCPms *float64
	SIms  *float64
	TTIms *float64
}

// FieldData is CrUX real-user field data.
type FieldData struct {
	OverallCategory string
	Metrics         map[string]FieldMetric
}

// FieldMetric is one CrUX metric percentile and its bucket.
type FieldMetric struct {
	Percentile float64
	Category   string
}

type rawResponse struct {
	LighthouseResult struct {
		Categories map[string]struct {
			Score *float64 `json:"score"`
		} `json:"categories"`
		Audits map[string]struct {
			NumericValue *float64 `json:"numericValue"`
		} `json:"audits"`
	} `json:"lighthouseResult"`
	LoadingExperience *struct {
		OverallCategory string `json:"overall_category"`
		Metrics         map[string]struct {
			Percentile float64 `json:"percentile"`
			Category   string  `json:"category"`
		} `json:"metrics"`
	} `json:"loadingExperience"`
}

func parseResult(data []byte) (*Result, error) {
	var raw rawResponse
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decoding PSI response: %w", err)
	}
	r := &Result{}
	r.Performance = score100(raw.LighthouseResult.Categories["performance"].Score)
	r.Accessibility = score100(raw.LighthouseResult.Categories["accessibility"].Score)
	r.BestPractices = score100(raw.LighthouseResult.Categories["best-practices"].Score)
	r.SEO = score100(raw.LighthouseResult.Categories["seo"].Score)

	audits := raw.LighthouseResult.Audits
	r.Lab = LabMetrics{
		LCPms: audits["largest-contentful-paint"].NumericValue,
		CLS:   audits["cumulative-layout-shift"].NumericValue,
		TBTms: audits["total-blocking-time"].NumericValue,
		FCPms: audits["first-contentful-paint"].NumericValue,
		SIms:  audits["speed-index"].NumericValue,
		TTIms: audits["interactive"].NumericValue,
	}

	if raw.LoadingExperience != nil {
		fd := &FieldData{
			OverallCategory: raw.LoadingExperience.OverallCategory,
			Metrics:         make(map[string]FieldMetric, len(raw.LoadingExperience.Metrics)),
		}
		for k, v := range raw.LoadingExperience.Metrics {
			fd.Metrics[k] = FieldMetric{Percentile: v.Percentile, Category: v.Category}
		}
		r.Field = fd
	}
	return r, nil
}

// score100 converts a 0-1 Lighthouse score to a rounded 0-100 value.
func score100(s *float64) *float64 {
	if s == nil {
		return nil
	}
	v := math.Round(*s * 100)
	return &v
}
