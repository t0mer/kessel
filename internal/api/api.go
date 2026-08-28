// Package api implements Kessel's REST API under /api/v1.
package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/t0mer/kessel/internal/crypto"
	"github.com/t0mer/kessel/internal/store"
)

// Runner runs all checks for a site (manual "run now").
type Runner interface {
	RunSite(ctx context.Context, site store.Site) ([]store.Run, error)
}

// ScheduleReloader refreshes the live scheduler after schedule changes.
type ScheduleReloader interface {
	Reload(ctx context.Context) error
}

// API holds the REST API dependencies.
type API struct {
	store      *store.Store
	runner     Runner
	reloader   ScheduleReloader
	reportsDir string
	cipher     *crypto.Cipher
	httpClient *http.Client
	log        *slog.Logger
}

// New builds an API. reportsDir bounds where run reports may be served from;
// cipher encrypts channel configs at rest.
func New(st *store.Store, r Runner, reloader ScheduleReloader, reportsDir string, cipher *crypto.Cipher, log *slog.Logger) *API {
	return &API{
		store:      st,
		runner:     r,
		reloader:   reloader,
		reportsDir: reportsDir,
		cipher:     cipher,
		httpClient: &http.Client{Timeout: 20 * time.Second},
		log:        log,
	}
}

func (a *API) reportsDirForTest() string { return a.reportsDir }

// Routes returns the API router (mounted under /api/v1 by the server).
func (a *API) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/sites", a.listSites)
	r.Post("/sites", a.createSite)
	r.Route("/sites/{siteID}", func(r chi.Router) {
		r.Get("/", a.getSite)
		r.Put("/", a.updateSite)
		r.Delete("/", a.deleteSite)
		r.Post("/run", a.runSiteNow)
		r.Get("/schedules", a.listSchedules)
		r.Post("/schedules", a.createSchedule)
		r.Get("/channels", a.getSiteChannels)
		r.Put("/channels", a.putSiteChannels)
	})
	r.Route("/schedules/{scheduleID}", func(r chi.Router) {
		r.Put("/", a.updateSchedule)
		r.Delete("/", a.deleteSchedule)
	})
	r.Get("/runs", a.listRuns)
	r.Get("/runs/{runID}", a.getRun)
	r.Get("/runs/{runID}/report", a.getRunReport)
	r.Get("/compare", a.compareRuns)

	r.Get("/channels", a.listChannels)
	r.Post("/channels", a.createChannel)
	r.Post("/channels/test", a.testChannel)
	r.Route("/channels/{channelID}", func(r chi.Router) {
		r.Put("/", a.updateChannel)
		r.Delete("/", a.deleteChannel)
	})

	r.Route("/sites/{siteID}/thresholds", func(r chi.Router) {
		r.Get("/", a.listThresholds)
		r.Post("/", a.createThreshold)
	})
	r.Route("/thresholds/{thresholdID}", func(r chi.Router) {
		r.Put("/", a.updateThreshold)
		r.Delete("/", a.deleteThreshold)
	})

	return r
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func idParam(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, name), 10, 64)
}
