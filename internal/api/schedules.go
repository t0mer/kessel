package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/t0mer/kessel/internal/scheduler"
	"github.com/t0mer/kessel/internal/store"
)

type scheduleResponse struct {
	ID        int64  `json:"id"`
	SiteID    int64  `json:"site_id"`
	CronExpr  string `json:"cron_expr"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
}

type scheduleRequest struct {
	CronExpr string `json:"cron_expr"`
	Enabled  *bool  `json:"enabled"`
}

func toScheduleResponse(s store.Schedule) scheduleResponse {
	return scheduleResponse{
		ID:        s.ID,
		SiteID:    s.SiteID,
		CronExpr:  s.CronExpr,
		Enabled:   s.Enabled,
		CreatedAt: s.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (a *API) reload(r *http.Request) {
	if err := a.reloader.Reload(r.Context()); err != nil {
		a.log.Error("scheduler reload", "error", err)
	}
}

func (a *API) listSchedules(w http.ResponseWriter, r *http.Request) {
	site, ok := a.loadSite(w, r)
	if !ok {
		return
	}
	scheds, err := a.store.ListSchedulesBySite(r.Context(), site.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]scheduleResponse, 0, len(scheds))
	for _, s := range scheds {
		out = append(out, toScheduleResponse(s))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) createSchedule(w http.ResponseWriter, r *http.Request) {
	site, ok := a.loadSite(w, r)
	if !ok {
		return
	}
	var req scheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := scheduler.ValidateCron(req.CronExpr); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sched, err := a.store.CreateSchedule(r.Context(), store.Schedule{SiteID: site.ID, CronExpr: req.CronExpr})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.reload(r)
	writeJSON(w, http.StatusCreated, toScheduleResponse(sched))
}

func (a *API) loadSchedule(w http.ResponseWriter, r *http.Request) (store.Schedule, bool) {
	id, err := idParam(r, "scheduleID")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid schedule id")
		return store.Schedule{}, false
	}
	sched, err := a.store.GetSchedule(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "schedule not found")
		return store.Schedule{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return store.Schedule{}, false
	}
	return sched, true
}

func (a *API) updateSchedule(w http.ResponseWriter, r *http.Request) {
	sched, ok := a.loadSchedule(w, r)
	if !ok {
		return
	}
	var req scheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := scheduler.ValidateCron(req.CronExpr); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sched.CronExpr = req.CronExpr
	if req.Enabled != nil {
		sched.Enabled = *req.Enabled
	}
	updated, err := a.store.UpdateSchedule(r.Context(), sched)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.reload(r)
	writeJSON(w, http.StatusOK, toScheduleResponse(updated))
}

func (a *API) deleteSchedule(w http.ResponseWriter, r *http.Request) {
	sched, ok := a.loadSchedule(w, r)
	if !ok {
		return
	}
	if err := a.store.DeleteSchedule(r.Context(), sched.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.reload(r)
	w.WriteHeader(http.StatusNoContent)
}
