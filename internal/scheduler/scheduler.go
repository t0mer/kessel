// Package scheduler runs site checks on cron schedules loaded from the store.
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/robfig/cron/v3"

	"github.com/t0mer/kessel/internal/store"
)

// SiteRunner runs all checks for a site.
type SiteRunner interface {
	RunSite(ctx context.Context, site store.Site) ([]store.Run, error)
}

// ValidateCron reports whether expr is a valid standard cron expression
// (5-field, plus @every and descriptors like @daily).
func ValidateCron(expr string) error {
	if _, err := cron.ParseStandard(expr); err != nil {
		return fmt.Errorf("invalid cron expression %q: %w", expr, err)
	}
	return nil
}

// Scheduler maps DB schedules onto cron jobs.
type Scheduler struct {
	store  *store.Store
	runner SiteRunner
	log    *slog.Logger
	cron   *cron.Cron

	mu      sync.Mutex
	entries map[int64]cron.EntryID // schedule ID -> cron entry
	ctx     context.Context
}

// NewScheduler builds a Scheduler (not yet started).
func NewScheduler(st *store.Store, r SiteRunner, log *slog.Logger) *Scheduler {
	return &Scheduler{
		store:   st,
		runner:  r,
		log:     log,
		cron:    cron.New(),
		entries: make(map[int64]cron.EntryID),
		ctx:     context.Background(),
	}
}

func (s *Scheduler) baseCtx() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ctx
}

// AddSchedule registers a cron entry for a schedule.
func (s *Scheduler) AddSchedule(sched store.Schedule) error {
	siteID := sched.SiteID
	id, err := s.cron.AddFunc(sched.CronExpr, func() {
		s.runSite(s.baseCtx(), siteID)
	})
	if err != nil {
		return fmt.Errorf("adding schedule %d (%q): %w", sched.ID, sched.CronExpr, err)
	}
	s.mu.Lock()
	s.entries[sched.ID] = id
	s.mu.Unlock()
	return nil
}

// RemoveSchedule unregisters a schedule's cron entry if present.
func (s *Scheduler) RemoveSchedule(scheduleID int64) {
	s.mu.Lock()
	id, ok := s.entries[scheduleID]
	if ok {
		delete(s.entries, scheduleID)
	}
	s.mu.Unlock()
	if ok {
		s.cron.Remove(id)
	}
}

// Reload rebuilds all cron entries from the enabled schedules in the store.
func (s *Scheduler) Reload(ctx context.Context) error {
	scheds, err := s.store.ListEnabledSchedules(ctx)
	if err != nil {
		return fmt.Errorf("loading schedules: %w", err)
	}
	s.mu.Lock()
	for _, id := range s.entries {
		s.cron.Remove(id)
	}
	s.entries = make(map[int64]cron.EntryID)
	s.mu.Unlock()

	for _, sc := range scheds {
		if err := s.AddSchedule(sc); err != nil {
			s.log.Error("registering schedule", "id", sc.ID, "error", err)
		}
	}
	return nil
}

// EntryCount returns the number of registered cron entries.
func (s *Scheduler) EntryCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entries)
}

// runSite executes a site's checks, re-fetching the site so disabled/edited
// sites are respected at fire time.
func (s *Scheduler) runSite(ctx context.Context, siteID int64) {
	site, err := s.store.GetSite(ctx, siteID)
	if err != nil {
		s.log.Error("scheduler: loading site", "site_id", siteID, "error", err)
		return
	}
	if !site.Enabled {
		s.log.Debug("scheduler: site disabled, skipping", "site", site.Slug)
		return
	}
	if _, err := s.runner.RunSite(ctx, site); err != nil {
		s.log.Error("scheduler: run site", "site", site.Slug, "error", err)
	}
}

// Start loads schedules and begins running them. ctx is used as the base
// context for every scheduled run (cancel it to stop in-flight runs on shutdown).
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	s.ctx = ctx
	s.mu.Unlock()
	if err := s.Reload(ctx); err != nil {
		return err
	}
	s.cron.Start()
	s.log.Info("scheduler started", "entries", s.EntryCount())
	return nil
}

// Stop halts the cron scheduler; the returned context is done when all
// currently-running jobs have finished.
func (s *Scheduler) Stop() context.Context {
	return s.cron.Stop()
}
