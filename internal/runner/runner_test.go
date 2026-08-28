package runner

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/t0mer/kessel/internal/psi"
	"github.com/t0mer/kessel/internal/store"
)

type fakePSI struct {
	result  *psi.Result
	raw     []byte
	err     error
	delay   time.Duration
	entered chan string   // if non-nil, receives strategy when Run starts
	release chan struct{} // if non-nil, Run blocks until a value is received/closed
	cur     int32
	max     int32
	mu      sync.Mutex
	calls   []string
}

func (f *fakePSI) Run(ctx context.Context, targetURL, strategy string) (*psi.Result, []byte, error) {
	n := atomic.AddInt32(&f.cur, 1)
	for {
		m := atomic.LoadInt32(&f.max)
		if n <= m || atomic.CompareAndSwapInt32(&f.max, m, n) {
			break
		}
	}
	defer atomic.AddInt32(&f.cur, -1)
	f.mu.Lock()
	f.calls = append(f.calls, strategy)
	f.mu.Unlock()
	if f.entered != nil {
		f.entered <- strategy
	}
	if f.release != nil {
		<-f.release
	}
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	return f.result, f.raw, f.err
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

func mkSite(t *testing.T, s *store.Store, strategy string) store.Site {
	t.Helper()
	site, err := s.CreateSite(context.Background(), store.Site{Name: "Site", URL: "https://ex.com", Strategy: strategy})
	if err != nil {
		t.Fatalf("CreateSite: %v", err)
	}
	return site
}

func f64(v float64) *float64 { return &v }

func sampleResult() *psi.Result {
	return &psi.Result{
		Performance:   f64(95),
		Accessibility: f64(90),
		BestPractices: f64(100),
		SEO:           f64(88),
		Lab:           psi.LabMetrics{LCPms: f64(1200), CLS: f64(0.02)},
	}
}

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestRunCheckSuccessPersistsMappedRun(t *testing.T) {
	s := newStore(t)
	site := mkSite(t, s, store.StrategyMobile)
	fp := &fakePSI{result: sampleResult(), raw: []byte(`{"ok":true}`)}
	r := NewRunner(s, fp, discardLogger(), Config{})

	run, err := r.RunCheck(context.Background(), site, store.StrategyMobile)
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if run.ID == 0 || run.Status != store.RunStatusSuccess {
		t.Fatalf("unexpected run: %+v", run)
	}
	if run.Perf == nil || *run.Perf != 95 {
		t.Errorf("Perf = %v, want 95", run.Perf)
	}
	if run.LCPms == nil || *run.LCPms != 1200 {
		t.Errorf("LCP = %v, want 1200", run.LCPms)
	}
	if len(run.RawJSONGz) == 0 {
		t.Error("expected gzipped raw JSON")
	}
	back, err := store.GunzipBytes(run.RawJSONGz)
	if err != nil || string(back) != `{"ok":true}` {
		t.Errorf("raw round-trip: %q err=%v", back, err)
	}
	if run.FinishedAt.Before(run.StartedAt) {
		t.Errorf("timestamps off: start=%v finish=%v", run.StartedAt, run.FinishedAt)
	}
}

func TestRunCheckFailureRecordsErrorRun(t *testing.T) {
	s := newStore(t)
	site := mkSite(t, s, store.StrategyDesktop)
	fp := &fakePSI{err: errors.New("PSI API returned status 429")}
	r := NewRunner(s, fp, discardLogger(), Config{})

	run, err := r.RunCheck(context.Background(), site, store.StrategyDesktop)
	if err != nil {
		t.Fatalf("RunCheck should not return the PSI error: %v", err)
	}
	if run.Status != store.RunStatusError || run.Error == "" {
		t.Fatalf("expected recorded error run, got %+v", run)
	}
	if run.Perf != nil {
		t.Errorf("expected nil scores on failure, got %v", run.Perf)
	}
	got, err := s.GetRun(context.Background(), run.ID)
	if err != nil || got.Status != store.RunStatusError {
		t.Errorf("failed run not persisted: %+v err=%v", got, err)
	}
}

func TestRunCheckInFlightGuard(t *testing.T) {
	s := newStore(t)
	site := mkSite(t, s, store.StrategyMobile)
	entered := make(chan string, 1)
	release := make(chan struct{})
	fp := &fakePSI{result: sampleResult(), raw: []byte(`{}`), entered: entered, release: release}
	r := NewRunner(s, fp, discardLogger(), Config{})

	go func() { _, _ = r.RunCheck(context.Background(), site, store.StrategyMobile) }()
	<-entered // first check is now inside PSI (in-flight registered)

	_, err := r.RunCheck(context.Background(), site, store.StrategyMobile)
	if !errors.Is(err, ErrInFlight) {
		t.Fatalf("second concurrent check err = %v, want ErrInFlight", err)
	}
	close(release)
}

func TestRunnerGlobalConcurrencyLimit(t *testing.T) {
	s := newStore(t)
	site := mkSite(t, s, store.StrategyBoth)
	fp := &fakePSI{result: sampleResult(), raw: []byte(`{}`), delay: 30 * time.Millisecond}
	r := NewRunner(s, fp, discardLogger(), Config{Concurrency: 1})

	var wg sync.WaitGroup
	for _, strat := range []string{store.StrategyMobile, store.StrategyDesktop} {
		wg.Add(1)
		go func(st string) {
			defer wg.Done()
			_, _ = r.RunCheck(context.Background(), site, st)
		}(strat)
	}
	wg.Wait()
	if got := atomic.LoadInt32(&fp.max); got != 1 {
		t.Fatalf("max concurrent PSI calls = %d, want 1", got)
	}
}

type fakeReporter struct {
	mu     sync.Mutex
	called int
}

func (f *fakeReporter) Render(site store.Site, run store.Run, rawJSON []byte) (string, error) {
	f.mu.Lock()
	f.called++
	f.mu.Unlock()
	return "/reports/x.html", nil
}

func TestRunCheckRendersReportOnSuccess(t *testing.T) {
	s := newStore(t)
	site := mkSite(t, s, store.StrategyMobile)
	fp := &fakePSI{result: sampleResult(), raw: []byte(`{}`)}
	rep := &fakeReporter{}
	r := NewRunner(s, fp, discardLogger(), Config{Reporter: rep})

	run, err := r.RunCheck(context.Background(), site, store.StrategyMobile)
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if run.ReportPath != "/reports/x.html" {
		t.Errorf("ReportPath = %q, want /reports/x.html", run.ReportPath)
	}
	got, _ := s.GetRun(context.Background(), run.ID)
	if got.ReportPath != "/reports/x.html" {
		t.Errorf("persisted ReportPath = %q", got.ReportPath)
	}
	if rep.called != 1 {
		t.Errorf("reporter called %d times, want 1", rep.called)
	}
}

func TestRunCheckNoReportOnFailure(t *testing.T) {
	s := newStore(t)
	site := mkSite(t, s, store.StrategyMobile)
	fp := &fakePSI{err: errors.New("boom")}
	rep := &fakeReporter{}
	r := NewRunner(s, fp, discardLogger(), Config{Reporter: rep})
	run, _ := r.RunCheck(context.Background(), site, store.StrategyMobile)
	if run.ReportPath != "" {
		t.Errorf("failed run should have no report, got %q", run.ReportPath)
	}
	if rep.called != 0 {
		t.Errorf("reporter should not run on failure, called %d", rep.called)
	}
}

type fakeNotifier struct {
	mu   sync.Mutex
	runs []int64
}

func (f *fakeNotifier) Notify(ctx context.Context, site store.Site, run store.Run) {
	f.mu.Lock()
	f.runs = append(f.runs, run.ID)
	f.mu.Unlock()
}

func TestRunCheckNotifies(t *testing.T) {
	s := newStore(t)
	site := mkSite(t, s, store.StrategyMobile)
	fp := &fakePSI{result: sampleResult(), raw: []byte(`{}`)}
	fn := &fakeNotifier{}
	r := NewRunner(s, fp, discardLogger(), Config{Notifier: fn})
	run, err := r.RunCheck(context.Background(), site, store.StrategyMobile)
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	fn.mu.Lock()
	defer fn.mu.Unlock()
	if len(fn.runs) != 1 || fn.runs[0] != run.ID {
		t.Fatalf("notifier not called for run %d: %v", run.ID, fn.runs)
	}
}

type fakeMetrics struct {
	mu   sync.Mutex
	runs int
}

func (f *fakeMetrics) ObserveRun(site store.Site, run store.Run, dur time.Duration) {
	f.mu.Lock()
	f.runs++
	f.mu.Unlock()
}

func TestRunCheckObservesMetrics(t *testing.T) {
	s := newStore(t)
	site := mkSite(t, s, store.StrategyMobile)
	fp := &fakePSI{result: sampleResult(), raw: []byte(`{}`)}
	fm := &fakeMetrics{}
	r := NewRunner(s, fp, discardLogger(), Config{Metrics: fm})
	if _, err := r.RunCheck(context.Background(), site, store.StrategyMobile); err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	fm.mu.Lock()
	defer fm.mu.Unlock()
	if fm.runs != 1 {
		t.Fatalf("ObserveRun called %d times, want 1", fm.runs)
	}
}

func TestStrategiesFor(t *testing.T) {
	if got := StrategiesFor(store.StrategyBoth); len(got) != 2 || got[0] != store.StrategyMobile || got[1] != store.StrategyDesktop {
		t.Errorf("both -> %v", got)
	}
	if got := StrategiesFor(store.StrategyMobile); len(got) != 1 || got[0] != store.StrategyMobile {
		t.Errorf("mobile -> %v", got)
	}
}

func TestRunSiteBothRunsTwice(t *testing.T) {
	s := newStore(t)
	site := mkSite(t, s, store.StrategyBoth)
	fp := &fakePSI{result: sampleResult(), raw: []byte(`{}`)}
	r := NewRunner(s, fp, discardLogger(), Config{})

	runs, err := r.RunSite(context.Background(), site)
	if err != nil {
		t.Fatalf("RunSite: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("got %d runs, want 2", len(runs))
	}
	fp.mu.Lock()
	defer fp.mu.Unlock()
	if len(fp.calls) != 2 || fp.calls[0] != store.StrategyMobile || fp.calls[1] != store.StrategyDesktop {
		t.Errorf("calls = %v, want [mobile desktop]", fp.calls)
	}
}

func TestRunSiteSingleStrategy(t *testing.T) {
	s := newStore(t)
	site := mkSite(t, s, store.StrategyDesktop)
	fp := &fakePSI{result: sampleResult(), raw: []byte(`{}`)}
	r := NewRunner(s, fp, discardLogger(), Config{})
	runs, err := r.RunSite(context.Background(), site)
	if err != nil {
		t.Fatalf("RunSite: %v", err)
	}
	if len(runs) != 1 || runs[0].Strategy != store.StrategyDesktop {
		t.Errorf("runs = %+v, want one desktop run", runs)
	}
}
