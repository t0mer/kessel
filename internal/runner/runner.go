// Package runner executes PSI checks with concurrency, in-flight, and spacing
// controls, and persists each result as a run.
package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/t0mer/kessel/internal/psi"
	"github.com/t0mer/kessel/internal/store"
)

// ErrInFlight is returned when a matching site+strategy check is already running.
var ErrInFlight = errors.New("check already in flight")

// PSIClient is the subset of the PSI client the runner needs.
type PSIClient interface {
	Run(ctx context.Context, targetURL, strategy string) (*psi.Result, []byte, error)
}

// Reporter renders a report for a completed run, returning its path.
type Reporter interface {
	Render(site store.Site, run store.Run, rawJSON []byte) (string, error)
}

// RunNotifier is notified after each completed run (best-effort).
type RunNotifier interface {
	Notify(ctx context.Context, site store.Site, run store.Run)
}

// Config configures a Runner.
type Config struct {
	Concurrency int
	MinSpacing  time.Duration
	Reporter    Reporter
	Notifier    RunNotifier
}

// Runner executes and persists PSI checks.
type Runner struct {
	store  *store.Store
	psi    PSIClient
	log    *slog.Logger
	sem      chan struct{}
	spacer   *spacer
	now      func() time.Time
	reporter Reporter
	notifier RunNotifier

	mu       sync.Mutex
	inflight map[string]struct{}
}

// NewRunner builds a Runner. Concurrency <= 0 defaults to 2.
func NewRunner(st *store.Store, client PSIClient, log *slog.Logger, cfg Config) *Runner {
	conc := cfg.Concurrency
	if conc <= 0 {
		conc = 2
	}
	now := time.Now
	return &Runner{
		store:    st,
		psi:      client,
		log:      log,
		sem:      make(chan struct{}, conc),
		spacer:   newSpacer(cfg.MinSpacing, now),
		now:      now,
		reporter: cfg.Reporter,
		notifier: cfg.Notifier,
		inflight: make(map[string]struct{}),
	}
}

func inflightKey(siteID int64, strategy string) string {
	return fmt.Sprintf("%d:%s", siteID, strategy)
}

func (r *Runner) acquireInflight(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, busy := r.inflight[key]; busy {
		return false
	}
	r.inflight[key] = struct{}{}
	return true
}

func (r *Runner) releaseInflight(key string) {
	r.mu.Lock()
	delete(r.inflight, key)
	r.mu.Unlock()
}

// RunCheck executes a single site+strategy check and persists the result.
// A PSI failure is recorded as a failed run (not returned as an error);
// ErrInFlight is returned if a matching check is already running.
func (r *Runner) RunCheck(ctx context.Context, site store.Site, strategy string) (store.Run, error) {
	key := inflightKey(site.ID, strategy)
	if !r.acquireInflight(key) {
		return store.Run{}, ErrInFlight
	}
	defer r.releaseInflight(key)

	select {
	case <-ctx.Done():
		return store.Run{}, ctx.Err()
	case r.sem <- struct{}{}:
	}
	defer func() { <-r.sem }()

	if err := r.spacer.wait(ctx); err != nil {
		return store.Run{}, err
	}

	startedAt := r.now()
	result, raw, psiErr := r.psi.Run(ctx, site.URL, strategy)
	finishedAt := r.now()

	run := store.Run{
		SiteID:     site.ID,
		Strategy:   strategy,
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
	}
	if psiErr != nil {
		run.Status = store.RunStatusError
		run.Error = psiErr.Error()
		r.log.Warn("psi check failed", "site", site.Slug, "strategy", strategy, "error", psiErr)
	} else {
		run.Status = store.RunStatusSuccess
		applyResult(&run, result)
	}
	if len(raw) > 0 {
		if gz, err := store.GzipBytes(raw); err != nil {
			r.log.Warn("gzip raw psi payload", "error", err)
		} else {
			run.RawJSONGz = gz
		}
	}

	stored, err := r.store.CreateRun(ctx, run)
	if err != nil {
		return store.Run{}, fmt.Errorf("persisting run: %w", err)
	}

	if r.reporter != nil && stored.Status == store.RunStatusSuccess && len(raw) > 0 {
		if path, rerr := r.reporter.Render(site, stored, raw); rerr != nil {
			r.log.Error("rendering report", "site", site.Slug, "strategy", strategy, "error", rerr)
		} else if serr := r.store.SetRunReportPath(ctx, stored.ID, path); serr != nil {
			r.log.Error("recording report path", "run", stored.ID, "error", serr)
		} else {
			stored.ReportPath = path
		}
	}

	if r.notifier != nil {
		r.notifier.Notify(ctx, site, stored)
	}
	return stored, nil
}

func applyResult(run *store.Run, res *psi.Result) {
	run.Perf = res.Performance
	run.Accessibility = res.Accessibility
	run.BestPractices = res.BestPractices
	run.SEO = res.SEO
	run.LCPms = res.Lab.LCPms
	run.CLS = res.Lab.CLS
	run.TBTms = res.Lab.TBTms
	run.FCPms = res.Lab.FCPms
	run.SIms = res.Lab.SIms
	run.TTIms = res.Lab.TTIms
}

// StrategiesFor expands a site's strategy setting into concrete strategies.
func StrategiesFor(siteStrategy string) []string {
	if siteStrategy == store.StrategyBoth {
		return []string{store.StrategyMobile, store.StrategyDesktop}
	}
	return []string{siteStrategy}
}

// RunSite runs every strategy configured for the site, sequentially.
// In-flight duplicates are skipped; the first real error stops the run.
func (r *Runner) RunSite(ctx context.Context, site store.Site) ([]store.Run, error) {
	var runs []store.Run
	for _, strat := range StrategiesFor(site.Strategy) {
		run, err := r.RunCheck(ctx, site, strat)
		if errors.Is(err, ErrInFlight) {
			r.log.Info("skipping in-flight check", "site", site.Slug, "strategy", strat)
			continue
		}
		if err != nil {
			return runs, err
		}
		runs = append(runs, run)
	}
	return runs, nil
}
