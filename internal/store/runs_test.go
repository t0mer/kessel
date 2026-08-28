package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func f64(v float64) *float64 { return &v }

func sampleRun(siteID int64) Run {
	return Run{
		SiteID:        siteID,
		Strategy:      StrategyMobile,
		StartedAt:     time.Unix(1000, 0),
		FinishedAt:    time.Unix(1005, 0),
		Status:        RunStatusSuccess,
		Perf:          f64(99),
		Accessibility: f64(88),
		BestPractices: f64(100),
		SEO:           f64(92),
		LCPms:         f64(1234.5),
		CLS:           f64(0.012),
		RawJSONGz:     []byte{0x1f, 0x8b, 0x00},
		ReportPath:    "/data/reports/s/mobile/1000.html",
	}
}

func TestCreateAndGetRun(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	site := mustSite(t, s)
	in := sampleRun(site.ID)
	created, err := s.CreateRun(ctx, in)
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	got, err := s.GetRun(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.Status != RunStatusSuccess || got.Strategy != StrategyMobile {
		t.Errorf("mismatch: %+v", got)
	}
	if got.Perf == nil || *got.Perf != 99 {
		t.Errorf("Perf = %v, want 99", got.Perf)
	}
	if got.CLS == nil || *got.CLS != 0.012 {
		t.Errorf("CLS = %v, want 0.012", got.CLS)
	}
	if got.StartedAt != time.Unix(1000, 0) || got.FinishedAt != time.Unix(1005, 0) {
		t.Errorf("timestamps: start=%v finish=%v", got.StartedAt, got.FinishedAt)
	}
	if len(got.RawJSONGz) != 3 {
		t.Errorf("raw gz len = %d, want 3", len(got.RawJSONGz))
	}
}

func TestCreateFailedRunNullMetrics(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	site := mustSite(t, s)
	in := Run{
		SiteID:     site.ID,
		Strategy:   StrategyDesktop,
		StartedAt:  time.Unix(2000, 0),
		FinishedAt: time.Unix(2001, 0),
		Status:     RunStatusError,
		Error:      "PSI API returned status 429",
	}
	created, err := s.CreateRun(ctx, in)
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	got, err := s.GetRun(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.Status != RunStatusError || got.Error == "" {
		t.Errorf("expected failed run with error, got %+v", got)
	}
	if got.Perf != nil || got.LCPms != nil {
		t.Errorf("expected nil metrics on failed run, got perf=%v lcp=%v", got.Perf, got.LCPms)
	}
}

func TestGetRunNotFound(t *testing.T) {
	s := openTemp(t)
	if _, err := s.GetRun(context.Background(), 424242); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func seedRuns(t *testing.T, s *Store, siteID int64) {
	t.Helper()
	ctx := context.Background()
	mk := func(strategy string, start int64, status string, perf float64) {
		r := Run{SiteID: siteID, Strategy: strategy, StartedAt: time.Unix(start, 0), FinishedAt: time.Unix(start+1, 0), Status: status, Perf: f64(perf)}
		if _, err := s.CreateRun(ctx, r); err != nil {
			t.Fatalf("seed CreateRun: %v", err)
		}
	}
	mk(StrategyMobile, 100, RunStatusSuccess, 80)
	mk(StrategyMobile, 200, RunStatusError, 0)
	mk(StrategyMobile, 300, RunStatusSuccess, 70)
	mk(StrategyDesktop, 150, RunStatusSuccess, 90)
}

func TestListRunsFilterAndOrder(t *testing.T) {
	s := openTemp(t)
	site := mustSite(t, s)
	seedRuns(t, s, site.ID)
	ctx := context.Background()

	all, err := s.ListRuns(ctx, RunFilter{SiteID: &site.ID})
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(all) != 4 {
		t.Fatalf("got %d runs, want 4", len(all))
	}
	if all[0].StartedAt.Before(all[1].StartedAt) {
		t.Errorf("not ordered newest-first: %v then %v", all[0].StartedAt, all[1].StartedAt)
	}

	mob, err := s.ListRuns(ctx, RunFilter{SiteID: &site.ID, Strategy: StrategyMobile})
	if err != nil {
		t.Fatalf("ListRuns mobile: %v", err)
	}
	if len(mob) != 3 {
		t.Fatalf("mobile runs = %d, want 3", len(mob))
	}
}

func TestListRunsPagination(t *testing.T) {
	s := openTemp(t)
	site := mustSite(t, s)
	seedRuns(t, s, site.ID)
	ctx := context.Background()
	page, err := s.ListRuns(ctx, RunFilter{SiteID: &site.ID, Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(page) != 2 {
		t.Fatalf("page size = %d, want 2", len(page))
	}
	total, err := s.CountRuns(ctx, RunFilter{SiteID: &site.ID})
	if err != nil {
		t.Fatalf("CountRuns: %v", err)
	}
	if total != 4 {
		t.Fatalf("total = %d, want 4", total)
	}
}

func TestLatestRun(t *testing.T) {
	s := openTemp(t)
	site := mustSite(t, s)
	seedRuns(t, s, site.ID)
	ctx := context.Background()
	latest, err := s.LatestRun(ctx, site.ID, StrategyMobile)
	if err != nil {
		t.Fatalf("LatestRun: %v", err)
	}
	if latest.StartedAt != time.Unix(300, 0) {
		t.Errorf("latest started = %v, want 300", latest.StartedAt)
	}
}

func TestPreviousSuccessfulRun(t *testing.T) {
	s := openTemp(t)
	site := mustSite(t, s)
	seedRuns(t, s, site.ID)
	ctx := context.Background()
	// Before the 300 run: newest *successful* mobile run before it is the 100 run
	// (200 is an error and must be skipped).
	prev, err := s.PreviousSuccessfulRun(ctx, site.ID, StrategyMobile, time.Unix(300, 0))
	if err != nil {
		t.Fatalf("PreviousSuccessfulRun: %v", err)
	}
	if prev.StartedAt != time.Unix(100, 0) {
		t.Errorf("prev started = %v, want 100", prev.StartedAt)
	}
}

func TestSetRunReportPath(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	site := mustSite(t, s)
	run, err := s.CreateRun(ctx, sampleRun(site.ID))
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if err := s.SetRunReportPath(ctx, run.ID, "/data/reports/s/mobile/x.html"); err != nil {
		t.Fatalf("SetRunReportPath: %v", err)
	}
	got, _ := s.GetRun(ctx, run.ID)
	if got.ReportPath != "/data/reports/s/mobile/x.html" {
		t.Errorf("report path = %q", got.ReportPath)
	}
}

func TestSetRunReportPathNotFound(t *testing.T) {
	s := openTemp(t)
	if err := s.SetRunReportPath(context.Background(), 99, "/x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestPreviousSuccessfulRunNone(t *testing.T) {
	s := openTemp(t)
	site := mustSite(t, s)
	seedRuns(t, s, site.ID)
	ctx := context.Background()
	_, err := s.PreviousSuccessfulRun(ctx, site.ID, StrategyMobile, time.Unix(100, 0))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
