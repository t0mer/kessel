// Package app is the Kessel composition root: it wires the store, PSI client,
// runner, scheduler, API, and HTTP server into a runnable application.
package app

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/t0mer/kessel/internal/api"
	"github.com/t0mer/kessel/internal/config"
	"github.com/t0mer/kessel/internal/psi"
	"github.com/t0mer/kessel/internal/runner"
	"github.com/t0mer/kessel/internal/scheduler"
	"github.com/t0mer/kessel/internal/server"
	"github.com/t0mer/kessel/internal/store"
)

// App is the assembled application.
type App struct {
	log       *slog.Logger
	store     *store.Store
	scheduler *scheduler.Scheduler
	server    *server.Server
}

// New builds the application from configuration and the embedded UI filesystem.
func New(cfg config.Config, log *slog.Logger, dist fs.FS) (*App, error) {
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating data dir %s: %w", cfg.DataDir, err)
	}
	dbPath := filepath.Join(cfg.DataDir, "kessel.db")
	st, err := store.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening store: %w", err)
	}

	if cfg.PSIAPIKey == "" {
		log.Warn("PSI API key not set; using keyless PageSpeed Insights (heavy rate limits)")
	}
	psiClient := psi.NewClient(psi.WithAPIKey(cfg.PSIAPIKey))
	rnr := runner.NewRunner(st, psiClient, log, runner.Config{Concurrency: cfg.PSIConcurrency})
	sch := scheduler.NewScheduler(st, rnr, log)
	restAPI := api.New(st, rnr, sch, log)

	srv := server.New(log, fmt.Sprintf(":%d", cfg.Port))
	srv.MountAPI(restAPI.Routes())
	srv.MountSPA(dist)

	return &App{log: log, store: st, scheduler: sch, server: srv}, nil
}

// Router exposes the HTTP handler (for tests).
func (a *App) Router() http.Handler { return a.server.Router() }

// Start starts background components (the scheduler).
func (a *App) Start(ctx context.Context) error {
	return a.scheduler.Start(ctx)
}

// Serve runs the HTTP server, blocking until it stops.
func (a *App) Serve() error { return a.server.Start() }

// Shutdown stops the scheduler, HTTP server, and closes the store.
func (a *App) Shutdown(ctx context.Context) error {
	stopCtx := a.scheduler.Stop()
	select {
	case <-stopCtx.Done():
	case <-ctx.Done():
	}
	srvErr := a.server.Shutdown(ctx)
	if err := a.store.Close(); err != nil {
		a.log.Error("closing store", "error", err)
	}
	return srvErr
}
