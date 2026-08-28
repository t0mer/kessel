package api

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/t0mer/kessel/internal/store"
)

func TestGetRunReportServesFile(t *testing.T) {
	a, s, _, _ := newAPI(t)
	site, _ := s.CreateSite(context.Background(), store.Site{Name: "S", URL: "https://s.com", Strategy: store.StrategyMobile})
	dir := a.reportsDirForTest()
	reportPath := filepath.Join(dir, "s", "mobile", "r.html")
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reportPath, []byte("<html>REPORT</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	run, _ := s.CreateRun(context.Background(), store.Run{SiteID: site.ID, Strategy: store.StrategyMobile, StartedAt: time.Unix(1, 0), FinishedAt: time.Unix(2, 0), Status: store.RunStatusSuccess, ReportPath: reportPath})

	rec := do(t, a, http.MethodGet, "/runs/"+itoa(run.ID)+"/report", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "<html>REPORT</html>" {
		t.Errorf("body = %q", rec.Body.String())
	}
}

func TestGetRunReportNoReport(t *testing.T) {
	a, s, _, _ := newAPI(t)
	site, _ := s.CreateSite(context.Background(), store.Site{Name: "S", URL: "https://s.com", Strategy: store.StrategyMobile})
	run, _ := s.CreateRun(context.Background(), store.Run{SiteID: site.ID, Strategy: store.StrategyMobile, StartedAt: time.Unix(1, 0), FinishedAt: time.Unix(2, 0), Status: store.RunStatusError})
	rec := do(t, a, http.MethodGet, "/runs/"+itoa(run.ID)+"/report", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestGetRunReportPathTraversalBlocked(t *testing.T) {
	a, s, _, _ := newAPI(t)
	site, _ := s.CreateSite(context.Background(), store.Site{Name: "S", URL: "https://s.com", Strategy: store.StrategyMobile})
	run, _ := s.CreateRun(context.Background(), store.Run{SiteID: site.ID, Strategy: store.StrategyMobile, StartedAt: time.Unix(1, 0), FinishedAt: time.Unix(2, 0), Status: store.RunStatusSuccess, ReportPath: "/etc/passwd"})
	rec := do(t, a, http.MethodGet, "/runs/"+itoa(run.ID)+"/report", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (traversal blocked)", rec.Code)
	}
}
