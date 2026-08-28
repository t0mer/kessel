package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/t0mer/kessel/internal/store"
)

type scores struct {
	Performance   *float64 `json:"performance"`
	Accessibility *float64 `json:"accessibility"`
	BestPractices *float64 `json:"best_practices"`
	SEO           *float64 `json:"seo"`
}

type metrics struct {
	LCPms *float64 `json:"lcp_ms"`
	CLS   *float64 `json:"cls"`
	TBTms *float64 `json:"tbt_ms"`
	FCPms *float64 `json:"fcp_ms"`
	SIms  *float64 `json:"si_ms"`
	TTIms *float64 `json:"tti_ms"`
}

type runResponse struct {
	ID         int64   `json:"id"`
	SiteID     int64   `json:"site_id"`
	Strategy   string  `json:"strategy"`
	Status     string  `json:"status"`
	StartedAt  string  `json:"started_at"`
	FinishedAt string  `json:"finished_at"`
	Scores     scores  `json:"scores"`
	Metrics    metrics `json:"metrics"`
	ReportPath string  `json:"report_path,omitempty"`
	Error      string  `json:"error,omitempty"`
}

type runsPage struct {
	Items  []runResponse `json:"items"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

type compareResponse struct {
	A            runResponse `json:"a"`
	B            runResponse `json:"b"`
	ScoreDeltas  scores      `json:"score_deltas"`
	MetricDeltas metrics     `json:"metric_deltas"`
}

func toRunResponse(r store.Run) runResponse {
	return runResponse{
		ID:         r.ID,
		SiteID:     r.SiteID,
		Strategy:   r.Strategy,
		Status:     r.Status,
		StartedAt:  r.StartedAt.UTC().Format(time.RFC3339),
		FinishedAt: r.FinishedAt.UTC().Format(time.RFC3339),
		Scores:     scores{r.Perf, r.Accessibility, r.BestPractices, r.SEO},
		Metrics:    metrics{r.LCPms, r.CLS, r.TBTms, r.FCPms, r.SIms, r.TTIms},
		ReportPath: r.ReportPath,
		Error:      r.Error,
	}
}

func (a *API) listRuns(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var f store.RunFilter
	if v := q.Get("site_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.SiteID = &id
		}
	}
	f.Strategy = q.Get("strategy")
	f.Limit = atoiDefault(q.Get("limit"), 50)
	f.Offset = atoiDefault(q.Get("offset"), 0)

	runs, err := a.store.ListRuns(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	total, err := a.store.CountRuns(r.Context(), f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]runResponse, 0, len(runs))
	for _, run := range runs {
		items = append(items, toRunResponse(run))
	}
	writeJSON(w, http.StatusOK, runsPage{Items: items, Total: total, Limit: f.Limit, Offset: f.Offset})
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return def
}

func (a *API) loadRun(w http.ResponseWriter, r *http.Request, param string) (store.Run, bool) {
	id, err := idParam(r, param)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid run id")
		return store.Run{}, false
	}
	run, err := a.store.GetRun(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "run not found")
		return store.Run{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return store.Run{}, false
	}
	return run, true
}

func (a *API) getRun(w http.ResponseWriter, r *http.Request) {
	run, ok := a.loadRun(w, r, "runID")
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, toRunResponse(run))
}

func (a *API) compareRuns(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	aID, err1 := strconv.ParseInt(q.Get("a"), 10, 64)
	bID, err2 := strconv.ParseInt(q.Get("b"), 10, 64)
	if err1 != nil || err2 != nil {
		writeError(w, http.StatusBadRequest, "query params a and b (run IDs) are required")
		return
	}
	runA, err := a.store.GetRun(r.Context(), aID)
	if err != nil {
		writeError(w, http.StatusNotFound, "run a not found")
		return
	}
	runB, err := a.store.GetRun(r.Context(), bID)
	if err != nil {
		writeError(w, http.StatusNotFound, "run b not found")
		return
	}
	writeJSON(w, http.StatusOK, compareResponse{
		A:           toRunResponse(runA),
		B:           toRunResponse(runB),
		ScoreDeltas: scores{delta(runA.Perf, runB.Perf), delta(runA.Accessibility, runB.Accessibility), delta(runA.BestPractices, runB.BestPractices), delta(runA.SEO, runB.SEO)},
		MetricDeltas: metrics{delta(runA.LCPms, runB.LCPms), delta(runA.CLS, runB.CLS), delta(runA.TBTms, runB.TBTms),
			delta(runA.FCPms, runB.FCPms), delta(runA.SIms, runB.SIms), delta(runA.TTIms, runB.TTIms)},
	})
}

// delta returns b-a when both are present, else nil.
func delta(a, b *float64) *float64 {
	if a == nil || b == nil {
		return nil
	}
	d := *b - *a
	return &d
}
