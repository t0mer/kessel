// Package report renders standalone HTML reports for PSI runs.
package report

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/t0mer/kessel/internal/store"
)

//go:embed templates/report.html.tmpl
var templatesFS embed.FS

// Renderer renders and persists run reports under a base directory.
type Renderer struct {
	dir  string
	tmpl *template.Template
}

// Opportunity is a Lighthouse improvement opportunity.
type Opportunity struct {
	Title        string
	DisplayValue string
	SavingsMs    float64
}

// NewRenderer parses the embedded template and prepares the reports directory.
func NewRenderer(reportsDir string) (*Renderer, error) {
	funcs := template.FuncMap{
		"dict":  dict,
		"score": scoreText,
		"cls":   scoreClass,
		"ms":    msText,
		"num":   numText,
	}
	tmpl, err := template.New("report.html.tmpl").Funcs(funcs).ParseFS(templatesFS, "templates/report.html.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parsing report template: %w", err)
	}
	return &Renderer{dir: reportsDir, tmpl: tmpl}, nil
}

// ReportPath returns the on-disk path for a report.
func ReportPath(reportsDir, slug, strategy string, ts time.Time) string {
	name := ts.UTC().Format("20060102T150405Z") + ".html"
	return filepath.Join(reportsDir, slug, strategy, name)
}

type templateData struct {
	Site          store.Site
	Run           store.Run
	Opportunities []Opportunity
	StartedAt     string
	GeneratedAt   string
}

// Render writes a self-contained HTML report and returns its path.
func (r *Renderer) Render(site store.Site, run store.Run, rawJSON []byte) (string, error) {
	path := ReportPath(r.dir, site.Slug, run.Strategy, run.StartedAt)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("creating report dir: %w", err)
	}
	data := templateData{
		Site:          site,
		Run:           run,
		Opportunities: extractOpportunities(rawJSON),
		StartedAt:     run.StartedAt.UTC().Format(time.RFC3339),
		GeneratedAt:   run.FinishedAt.UTC().Format(time.RFC3339),
	}
	var buf bytes.Buffer
	if err := r.tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering report: %w", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return "", fmt.Errorf("writing report: %w", err)
	}
	return path, nil
}

func extractOpportunities(rawJSON []byte) []Opportunity {
	var doc struct {
		LighthouseResult struct {
			Audits map[string]struct {
				Title        string `json:"title"`
				DisplayValue string `json:"displayValue"`
				Details      struct {
					Type             string  `json:"type"`
					OverallSavingsMs float64 `json:"overallSavingsMs"`
				} `json:"details"`
			} `json:"audits"`
		} `json:"lighthouseResult"`
	}
	if err := json.Unmarshal(rawJSON, &doc); err != nil {
		return nil
	}
	var out []Opportunity
	for _, a := range doc.LighthouseResult.Audits {
		if a.Details.Type == "opportunity" && a.Details.OverallSavingsMs > 0 {
			out = append(out, Opportunity{Title: a.Title, DisplayValue: a.DisplayValue, SavingsMs: a.Details.OverallSavingsMs})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SavingsMs > out[j].SavingsMs })
	return out
}

// --- template helpers ---

func dict(kv ...any) map[string]any {
	m := make(map[string]any, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		m[fmt.Sprint(kv[i])] = kv[i+1]
	}
	return m
}

func scoreText(v *float64) string {
	if v == nil {
		return "—"
	}
	return fmt.Sprintf("%.0f", *v)
}

func scoreClass(v *float64) string {
	if v == nil {
		return "na"
	}
	switch {
	case *v >= 90:
		return "good"
	case *v >= 50:
		return "avg"
	default:
		return "poor"
	}
}

func msText(v *float64) string {
	if v == nil {
		return "—"
	}
	return fmt.Sprintf("%.0f ms", *v)
}

func numText(v *float64) string {
	if v == nil {
		return "—"
	}
	return fmt.Sprintf("%.3f", *v)
}
