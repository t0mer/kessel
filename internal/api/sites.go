package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/t0mer/kessel/internal/store"
)

type siteResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	URL       string `json:"url"`
	Strategy  string `json:"strategy"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type siteRequest struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Strategy string `json:"strategy"`
	Enabled  *bool  `json:"enabled"`
}

func toSiteResponse(s store.Site) siteResponse {
	return siteResponse{
		ID:        s.ID,
		Name:      s.Name,
		Slug:      s.Slug,
		URL:       s.URL,
		Strategy:  s.Strategy,
		Enabled:   s.Enabled,
		CreatedAt: s.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: s.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func validStrategy(s string) bool {
	return s == store.StrategyMobile || s == store.StrategyDesktop || s == store.StrategyBoth
}

func validateSite(req siteRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("name is required")
	}
	u, err := url.Parse(req.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return errors.New("url must be a valid http or https URL")
	}
	if !validStrategy(req.Strategy) {
		return errors.New("strategy must be mobile, desktop, or both")
	}
	return nil
}

func (a *API) listSites(w http.ResponseWriter, r *http.Request) {
	sites, err := a.store.ListSites(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]siteResponse, 0, len(sites))
	for _, s := range sites {
		out = append(out, toSiteResponse(s))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) createSite(w http.ResponseWriter, r *http.Request) {
	var req siteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := validateSite(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	site, err := a.store.CreateSite(r.Context(), store.Site{Name: req.Name, URL: req.URL, Strategy: req.Strategy})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toSiteResponse(site))
}

func (a *API) loadSite(w http.ResponseWriter, r *http.Request) (store.Site, bool) {
	id, err := idParam(r, "siteID")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid site id")
		return store.Site{}, false
	}
	site, err := a.store.GetSite(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "site not found")
		return store.Site{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return store.Site{}, false
	}
	return site, true
}

func (a *API) getSite(w http.ResponseWriter, r *http.Request) {
	site, ok := a.loadSite(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, toSiteResponse(site))
}

func (a *API) updateSite(w http.ResponseWriter, r *http.Request) {
	site, ok := a.loadSite(w, r)
	if !ok {
		return
	}
	var req siteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := validateSite(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	site.Name = req.Name
	site.URL = req.URL
	site.Strategy = req.Strategy
	if req.Enabled != nil {
		site.Enabled = *req.Enabled
	}
	updated, err := a.store.UpdateSite(r.Context(), site)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toSiteResponse(updated))
}

func (a *API) deleteSite(w http.ResponseWriter, r *http.Request) {
	site, ok := a.loadSite(w, r)
	if !ok {
		return
	}
	if err := a.store.DeleteSite(r.Context(), site.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) runSiteNow(w http.ResponseWriter, r *http.Request) {
	site, ok := a.loadSite(w, r)
	if !ok {
		return
	}
	go func() {
		if _, err := a.runner.RunSite(context.Background(), site); err != nil {
			a.log.Error("manual run failed", "site", site.Slug, "error", err)
		}
	}()
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "started", "site": site.Slug})
}
