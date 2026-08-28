package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/t0mer/kessel/internal/notify"
	"github.com/t0mer/kessel/internal/store"
)

type thresholdResponse struct {
	ID       int64   `json:"id"`
	SiteID   int64   `json:"site_id"`
	Category string  `json:"category"`
	Mode     string  `json:"mode"`
	Value    float64 `json:"value"`
	Enabled  bool    `json:"enabled"`
}

type thresholdRequest struct {
	Category string  `json:"category"`
	Mode     string  `json:"mode"`
	Value    float64 `json:"value"`
	Enabled  *bool   `json:"enabled"`
}

func toThresholdResponse(r store.ThresholdRule) thresholdResponse {
	return thresholdResponse{ID: r.ID, SiteID: r.SiteID, Category: r.Category, Mode: r.Mode, Value: r.Value, Enabled: r.Enabled}
}

func validCategory(c string) bool {
	switch c {
	case notify.CategoryPerformance, notify.CategoryAccessibility, notify.CategoryBestPractices, notify.CategorySEO:
		return true
	}
	return false
}

func validMode(m string) bool { return m == store.ThresholdAbsolute || m == store.ThresholdDelta }

func validateThreshold(req thresholdRequest) error {
	if !validCategory(req.Category) {
		return errors.New("category must be performance, accessibility, best_practices, or seo")
	}
	if !validMode(req.Mode) {
		return errors.New("mode must be absolute or delta")
	}
	return nil
}

func (a *API) listThresholds(w http.ResponseWriter, r *http.Request) {
	site, ok := a.loadSite(w, r)
	if !ok {
		return
	}
	rules, err := a.store.ListThresholdRulesBySite(r.Context(), site.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]thresholdResponse, 0, len(rules))
	for _, rl := range rules {
		out = append(out, toThresholdResponse(rl))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) createThreshold(w http.ResponseWriter, r *http.Request) {
	site, ok := a.loadSite(w, r)
	if !ok {
		return
	}
	var req thresholdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := validateThreshold(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	rule, err := a.store.CreateThresholdRule(r.Context(), store.ThresholdRule{SiteID: site.ID, Category: req.Category, Mode: req.Mode, Value: req.Value})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toThresholdResponse(rule))
}

func (a *API) loadThreshold(w http.ResponseWriter, r *http.Request) (store.ThresholdRule, bool) {
	id, err := idParam(r, "thresholdID")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid threshold id")
		return store.ThresholdRule{}, false
	}
	rule, err := a.store.GetThresholdRule(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "threshold rule not found")
		return store.ThresholdRule{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return store.ThresholdRule{}, false
	}
	return rule, true
}

func (a *API) updateThreshold(w http.ResponseWriter, r *http.Request) {
	rule, ok := a.loadThreshold(w, r)
	if !ok {
		return
	}
	var req thresholdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := validateThreshold(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	rule.Category = req.Category
	rule.Mode = req.Mode
	rule.Value = req.Value
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	updated, err := a.store.UpdateThresholdRule(r.Context(), rule)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toThresholdResponse(updated))
}

func (a *API) deleteThreshold(w http.ResponseWriter, r *http.Request) {
	rule, ok := a.loadThreshold(w, r)
	if !ok {
		return
	}
	if err := a.store.DeleteThresholdRule(r.Context(), rule.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
