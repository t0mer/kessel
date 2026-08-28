package scheduler

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"sync"
	"testing"

	"github.com/t0mer/kessel/internal/store"
)

type fakeRunner struct {
	mu    sync.Mutex
	sites []int64
}

func (f *fakeRunner) RunSite(ctx context.Context, site store.Site) ([]store.Run, error) {
	f.mu.Lock()
	f.sites = append(f.sites, site.ID)
	f.mu.Unlock()
	return nil, nil
}

func (f *fakeRunner) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.sites)
}

func newStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "k.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestValidateCron(t *testing.T) {
	valid := []string{"@every 6h", "0 * * * *", "*/5 * * * *", "@daily"}
	for _, e := range valid {
		if err := ValidateCron(e); err != nil {
			t.Errorf("ValidateCron(%q) = %v, want nil", e, err)
		}
	}
	invalid := []string{"", "not a cron", "* * * *", "99 * * * *"}
	for _, e := range invalid {
		if err := ValidateCron(e); err == nil {
			t.Errorf("ValidateCron(%q) = nil, want error", e)
		}
	}
}

func TestReloadRegistersEnabledSchedules(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	site, _ := s.CreateSite(ctx, store.Site{Name: "S", URL: "https://s.com", Strategy: store.StrategyMobile})
	on1, _ := s.CreateSchedule(ctx, store.Schedule{SiteID: site.ID, CronExpr: "@every 6h"})
	_, _ = s.CreateSchedule(ctx, store.Schedule{SiteID: site.ID, CronExpr: "0 * * * *"})
	off, _ := s.CreateSchedule(ctx, store.Schedule{SiteID: site.ID, CronExpr: "@every 12h"})
	if err := s.SetScheduleEnabled(ctx, off.ID, false); err != nil {
		t.Fatalf("disable: %v", err)
	}
	_ = on1

	sched := NewScheduler(s, &fakeRunner{}, discardLogger())
	if err := sched.Reload(ctx); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if sched.EntryCount() != 2 {
		t.Fatalf("EntryCount = %d, want 2 (only enabled)", sched.EntryCount())
	}
}

func TestAddRemoveSchedule(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	site, _ := s.CreateSite(ctx, store.Site{Name: "S", URL: "https://s.com", Strategy: store.StrategyMobile})
	sc, _ := s.CreateSchedule(ctx, store.Schedule{SiteID: site.ID, CronExpr: "@every 6h"})

	sched := NewScheduler(s, &fakeRunner{}, discardLogger())
	if err := sched.AddSchedule(sc); err != nil {
		t.Fatalf("AddSchedule: %v", err)
	}
	if sched.EntryCount() != 1 {
		t.Fatalf("after add EntryCount = %d, want 1", sched.EntryCount())
	}
	sched.RemoveSchedule(sc.ID)
	if sched.EntryCount() != 0 {
		t.Fatalf("after remove EntryCount = %d, want 0", sched.EntryCount())
	}
}

func TestAddScheduleInvalidCron(t *testing.T) {
	s := newStore(t)
	sched := NewScheduler(s, &fakeRunner{}, discardLogger())
	if err := sched.AddSchedule(store.Schedule{ID: 1, SiteID: 1, CronExpr: "nonsense"}); err == nil {
		t.Fatal("expected error for invalid cron expr")
	}
}

func TestRunSiteInvokesRunner(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	site, _ := s.CreateSite(ctx, store.Site{Name: "S", URL: "https://s.com", Strategy: store.StrategyMobile})
	fr := &fakeRunner{}
	sched := NewScheduler(s, fr, discardLogger())
	sched.runSite(ctx, site.ID)
	if fr.count() != 1 {
		t.Fatalf("runner calls = %d, want 1", fr.count())
	}
}

func TestRunSiteSkipsDisabledSite(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	site, _ := s.CreateSite(ctx, store.Site{Name: "S", URL: "https://s.com", Strategy: store.StrategyMobile})
	if err := s.SetSiteEnabled(ctx, site.ID, false); err != nil {
		t.Fatalf("disable site: %v", err)
	}
	fr := &fakeRunner{}
	sched := NewScheduler(s, fr, discardLogger())
	sched.runSite(ctx, site.ID)
	if fr.count() != 0 {
		t.Fatalf("runner calls = %d, want 0 (site disabled)", fr.count())
	}
}

func TestRunSiteMissingSiteNoPanic(t *testing.T) {
	s := newStore(t)
	fr := &fakeRunner{}
	sched := NewScheduler(s, fr, discardLogger())
	sched.runSite(context.Background(), 987654) // no such site: must not panic or call runner
	if fr.count() != 0 {
		t.Fatalf("runner calls = %d, want 0", fr.count())
	}
}

func TestStartStop(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	site, _ := s.CreateSite(ctx, store.Site{Name: "S", URL: "https://s.com", Strategy: store.StrategyMobile})
	_, _ = s.CreateSchedule(ctx, store.Schedule{SiteID: site.ID, CronExpr: "@every 6h"})
	sched := NewScheduler(s, &fakeRunner{}, discardLogger())
	if err := sched.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if sched.EntryCount() != 1 {
		t.Fatalf("EntryCount after Start = %d, want 1", sched.EntryCount())
	}
	stopCtx := sched.Stop()
	<-stopCtx.Done() // completes when jobs drained
}
