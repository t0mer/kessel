package api

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/t0mer/kessel/internal/store"
)

func itoa(v int64) string { return strconv.FormatInt(v, 10) }

func seedRun(t *testing.T, s *store.Store, siteID int64, strategy string, start int64, perf float64) store.Run {
	t.Helper()
	p := perf
	run, err := s.CreateRun(context.Background(), store.Run{
		SiteID: siteID, Strategy: strategy,
		StartedAt: time.Unix(start, 0), FinishedAt: time.Unix(start+2, 0),
		Status: store.RunStatusSuccess, Perf: &p,
	})
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	return run
}

func TestListRunsFilterPaginate(t *testing.T) {
	a, s, _, _ := newAPI(t)
	site, _ := s.CreateSite(context.Background(), store.Site{Name: "S", URL: "https://s.com", Strategy: store.StrategyMobile})
	seedRun(t, s, site.ID, store.StrategyMobile, 100, 80)
	seedRun(t, s, site.ID, store.StrategyMobile, 200, 70)
	seedRun(t, s, site.ID, store.StrategyDesktop, 150, 90)

	rec := do(t, a, http.MethodGet, "/runs?site_id=1&strategy=mobile&limit=1&offset=0", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var page runsPage
	decode(t, rec, &page)
	if page.Total != 2 {
		t.Errorf("total = %d, want 2 (mobile only)", page.Total)
	}
	if len(page.Items) != 1 {
		t.Errorf("items = %d, want 1 (limit)", len(page.Items))
	}
	if page.Items[0].StartedAt == "" || page.Items[0].Scores.Performance == nil {
		t.Errorf("run response incomplete: %+v", page.Items[0])
	}
}

func TestGetRun(t *testing.T) {
	a, s, _, _ := newAPI(t)
	site, _ := s.CreateSite(context.Background(), store.Site{Name: "S", URL: "https://s.com", Strategy: store.StrategyMobile})
	run := seedRun(t, s, site.ID, store.StrategyMobile, 100, 80)
	rec := do(t, a, http.MethodGet, "/runs/"+itoa(run.ID), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got runResponse
	decode(t, rec, &got)
	if got.ID != run.ID {
		t.Errorf("id = %d, want %d", got.ID, run.ID)
	}
}

func TestGetRunNotFound(t *testing.T) {
	a, _, _, _ := newAPI(t)
	rec := do(t, a, http.MethodGet, "/runs/999", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestCompareRuns(t *testing.T) {
	a, s, _, _ := newAPI(t)
	site, _ := s.CreateSite(context.Background(), store.Site{Name: "S", URL: "https://s.com", Strategy: store.StrategyMobile})
	r1 := seedRun(t, s, site.ID, store.StrategyMobile, 100, 80)
	r2 := seedRun(t, s, site.ID, store.StrategyMobile, 200, 90)
	rec := do(t, a, http.MethodGet, "/compare?a="+itoa(r1.ID)+"&b="+itoa(r2.ID), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	var cmp compareResponse
	decode(t, rec, &cmp)
	if cmp.ScoreDeltas.Performance == nil || *cmp.ScoreDeltas.Performance != 10 {
		t.Errorf("perf delta = %v, want 10", cmp.ScoreDeltas.Performance)
	}
}

func TestCompareRunsMissingParam(t *testing.T) {
	a, _, _, _ := newAPI(t)
	rec := do(t, a, http.MethodGet, "/compare?a=1", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
