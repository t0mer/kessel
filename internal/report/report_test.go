package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/t0mer/kessel/internal/store"
)

func f64(v float64) *float64 { return &v }

func TestRenderWritesSelfContainedHTML(t *testing.T) {
	dir := t.TempDir()
	r, err := NewRenderer(dir)
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}
	site := store.Site{Name: "My Site", Slug: "my-site", URL: "https://ex.com"}
	run := store.Run{
		Strategy: store.StrategyMobile, Status: store.RunStatusSuccess,
		StartedAt: time.Unix(1_700_000_000, 0),
		Perf:      f64(95), Accessibility: f64(88), BestPractices: f64(100), SEO: f64(92),
		LCPms:     f64(1200), CLS: f64(0.02),
	}
	raw := []byte(`{"lighthouseResult":{"audits":{
		"uses-webp":{"title":"Serve images in WebP","displayValue":"1.2s","details":{"type":"opportunity","overallSavingsMs":1200}},
		"render-blocking":{"title":"Eliminate render-blocking","displayValue":"","details":{"type":"opportunity","overallSavingsMs":300}}
	}}}`)

	path, err := r.Render(site, run, raw)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.HasPrefix(path, dir) {
		t.Errorf("path %q not under %q", path, dir)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading report: %v", err)
	}
	html := string(data)
	for _, want := range []string{"My Site", "https://ex.com", "95", "Serve images in WebP", "Core Web Vitals"} {
		if !strings.Contains(html, want) {
			t.Errorf("report missing %q", want)
		}
	}
	if strings.Contains(html, "http://") && strings.Contains(html, "<link") {
		t.Error("report should be self-contained (no external stylesheets)")
	}
	if strings.Index(html, "Serve images in WebP") > strings.Index(html, "Eliminate render-blocking") {
		t.Error("opportunities not sorted by savings desc")
	}
	if filepath.Base(filepath.Dir(path)) != "mobile" {
		t.Errorf("strategy dir wrong: %q", path)
	}
}

func TestRenderNilMetricsShowNA(t *testing.T) {
	dir := t.TempDir()
	r, _ := NewRenderer(dir)
	site := store.Site{Name: "S", Slug: "s", URL: "https://s.com"}
	run := store.Run{Strategy: store.StrategyDesktop, Status: store.RunStatusSuccess, StartedAt: time.Unix(100, 0)}
	path, err := r.Render(site, run, []byte(`{}`))
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "—") {
		t.Error("expected em-dash for missing metrics/scores")
	}
}
