package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/t0mer/kessel/internal/store"
)

func TestCreateThresholdValidation(t *testing.T) {
	a, s, _, _ := newAPI(t)
	_, _ = s.CreateSite(context.Background(), store.Site{Name: "S", URL: "https://s.com", Strategy: store.StrategyMobile})
	bad := []string{
		`{"category":"bogus","mode":"absolute","value":80}`,
		`{"category":"performance","mode":"bogus","value":80}`,
	}
	for _, body := range bad {
		if rec := do(t, a, http.MethodPost, "/sites/1/thresholds", body); rec.Code != http.StatusBadRequest {
			t.Errorf("body %s -> %d, want 400", body, rec.Code)
		}
	}
}

func TestThresholdCRUD(t *testing.T) {
	a, s, _, _ := newAPI(t)
	_, _ = s.CreateSite(context.Background(), store.Site{Name: "S", URL: "https://s.com", Strategy: store.StrategyMobile})
	rec := do(t, a, http.MethodPost, "/sites/1/thresholds", `{"category":"performance","mode":"absolute","value":80}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create -> %d, want 201 (body=%s)", rec.Code, rec.Body.String())
	}
	var created thresholdResponse
	decode(t, rec, &created)
	if created.ID == 0 || created.Category != "performance" || !created.Enabled {
		t.Fatalf("unexpected: %+v", created)
	}

	list := do(t, a, http.MethodGet, "/sites/1/thresholds", "")
	var rules []thresholdResponse
	decode(t, list, &rules)
	if len(rules) != 1 {
		t.Fatalf("list = %d, want 1", len(rules))
	}

	upd := do(t, a, http.MethodPut, "/thresholds/1", `{"category":"seo","mode":"delta","value":10,"enabled":false}`)
	if upd.Code != http.StatusOK {
		t.Fatalf("update -> %d (body=%s)", upd.Code, upd.Body.String())
	}
	var updated thresholdResponse
	decode(t, upd, &updated)
	if updated.Category != "seo" || updated.Mode != "delta" || updated.Enabled {
		t.Fatalf("update not applied: %+v", updated)
	}

	if del := do(t, a, http.MethodDelete, "/thresholds/1", ""); del.Code != http.StatusNoContent {
		t.Fatalf("delete -> %d, want 204", del.Code)
	}
}
