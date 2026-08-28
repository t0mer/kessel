package psi

import (
	"os"
	"testing"
)

func loadFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/response.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	return data
}

func TestParseResultCategories(t *testing.T) {
	r, err := parseResult(loadFixture(t))
	if err != nil {
		t.Fatalf("parseResult: %v", err)
	}
	if r.Performance == nil || *r.Performance != 99 {
		t.Errorf("Performance = %v, want 99", r.Performance)
	}
	if r.Accessibility == nil || *r.Accessibility != 88 {
		t.Errorf("Accessibility = %v, want 88", r.Accessibility)
	}
	if r.BestPractices == nil || *r.BestPractices != 100 {
		t.Errorf("BestPractices = %v, want 100", r.BestPractices)
	}
	if r.SEO == nil || *r.SEO != 92 {
		t.Errorf("SEO = %v, want 92", r.SEO)
	}
}

func TestParseResultLabMetrics(t *testing.T) {
	r, err := parseResult(loadFixture(t))
	if err != nil {
		t.Fatalf("parseResult: %v", err)
	}
	if r.Lab.LCPms == nil || *r.Lab.LCPms != 1234.5 {
		t.Errorf("LCP = %v, want 1234.5", r.Lab.LCPms)
	}
	if r.Lab.CLS == nil || *r.Lab.CLS != 0.012 {
		t.Errorf("CLS = %v, want 0.012", r.Lab.CLS)
	}
	if r.Lab.TBTms == nil || *r.Lab.TBTms != 50 {
		t.Errorf("TBT = %v, want 50", r.Lab.TBTms)
	}
	if r.Lab.FCPms == nil || *r.Lab.FCPms != 900 {
		t.Errorf("FCP = %v, want 900", r.Lab.FCPms)
	}
	if r.Lab.SIms == nil || *r.Lab.SIms != 2000 {
		t.Errorf("SI = %v, want 2000", r.Lab.SIms)
	}
	if r.Lab.TTIms == nil || *r.Lab.TTIms != 3000 {
		t.Errorf("TTI = %v, want 3000", r.Lab.TTIms)
	}
}

func TestParseResultFieldData(t *testing.T) {
	r, err := parseResult(loadFixture(t))
	if err != nil {
		t.Fatalf("parseResult: %v", err)
	}
	if r.Field == nil {
		t.Fatal("expected field data")
	}
	if r.Field.OverallCategory != "AVERAGE" {
		t.Errorf("overall = %q, want AVERAGE", r.Field.OverallCategory)
	}
	m, ok := r.Field.Metrics["LARGEST_CONTENTFUL_PAINT_MS"]
	if !ok || m.Percentile != 2500 {
		t.Errorf("LCP field metric = %+v, want percentile 2500", m)
	}
}

func TestParseResultNoFieldData(t *testing.T) {
	// A response without loadingExperience yields nil Field, no error.
	data := []byte(`{"lighthouseResult":{"categories":{"performance":{"score":0.5}},"audits":{}}}`)
	r, err := parseResult(data)
	if err != nil {
		t.Fatalf("parseResult: %v", err)
	}
	if r.Field != nil {
		t.Errorf("expected nil Field, got %+v", r.Field)
	}
	if r.Performance == nil || *r.Performance != 50 {
		t.Errorf("Performance = %v, want 50", r.Performance)
	}
	if r.Lab.LCPms != nil {
		t.Errorf("expected nil LCP when audit absent, got %v", r.Lab.LCPms)
	}
}

func TestParseResultInvalidJSON(t *testing.T) {
	if _, err := parseResult([]byte(`not json`)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
