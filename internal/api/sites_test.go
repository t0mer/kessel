package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/t0mer/kessel/internal/store"
)

func TestCreateSiteValidation(t *testing.T) {
	a, _, _, _ := newAPI(t)
	bad := []string{
		`{"name":"","url":"https://x.com","strategy":"mobile"}`,
		`{"name":"X","url":"ftp://x.com","strategy":"mobile"}`,
		`{"name":"X","url":"not a url","strategy":"mobile"}`,
		`{"name":"X","url":"https://x.com","strategy":"phone"}`,
	}
	for _, body := range bad {
		rec := do(t, a, http.MethodPost, "/sites", body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("body %s -> %d, want 400", body, rec.Code)
		}
	}
}

func TestCreateAndGetSite(t *testing.T) {
	a, _, _, _ := newAPI(t)
	rec := do(t, a, http.MethodPost, "/sites", `{"name":"My Site","url":"https://ex.com","strategy":"both"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201 (body=%s)", rec.Code, rec.Body.String())
	}
	var created siteResponse
	decode(t, rec, &created)
	if created.ID == 0 || created.Slug != "my-site" || !created.Enabled {
		t.Fatalf("unexpected created site: %+v", created)
	}
	if _, err := time.Parse(time.RFC3339, created.CreatedAt); err != nil {
		t.Errorf("CreatedAt not RFC3339: %q", created.CreatedAt)
	}

	rec = do(t, a, http.MethodGet, "/sites/1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200", rec.Code)
	}
}

func TestGetSiteNotFound(t *testing.T) {
	a, _, _, _ := newAPI(t)
	rec := do(t, a, http.MethodGet, "/sites/999", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestListSites(t *testing.T) {
	a, s, _, _ := newAPI(t)
	_, _ = s.CreateSite(context.Background(), store.Site{Name: "A", URL: "https://a.com", Strategy: store.StrategyMobile})
	_, _ = s.CreateSite(context.Background(), store.Site{Name: "B", URL: "https://b.com", Strategy: store.StrategyMobile})
	rec := do(t, a, http.MethodGet, "/sites", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var list []siteResponse
	decode(t, rec, &list)
	if len(list) != 2 {
		t.Fatalf("got %d sites, want 2", len(list))
	}
}

func TestUpdateSite(t *testing.T) {
	a, s, _, _ := newAPI(t)
	site, _ := s.CreateSite(context.Background(), store.Site{Name: "Orig", URL: "https://o.com", Strategy: store.StrategyMobile})
	body := `{"name":"New","url":"https://new.com","strategy":"desktop","enabled":false}`
	rec := do(t, a, http.MethodPut, "/sites/1", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	var updated siteResponse
	decode(t, rec, &updated)
	if updated.Name != "New" || updated.Strategy != "desktop" || updated.Enabled {
		t.Fatalf("update not applied: %+v", updated)
	}
	_ = site
}

func TestDeleteSite(t *testing.T) {
	a, s, _, _ := newAPI(t)
	_, _ = s.CreateSite(context.Background(), store.Site{Name: "Del", URL: "https://d.com", Strategy: store.StrategyMobile})
	rec := do(t, a, http.MethodDelete, "/sites/1", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	rec = do(t, a, http.MethodGet, "/sites/1", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("after delete get = %d, want 404", rec.Code)
	}
}

func TestRunSiteNow(t *testing.T) {
	a, s, fr, _ := newAPI(t)
	site, _ := s.CreateSite(context.Background(), store.Site{Name: "R", URL: "https://r.com", Strategy: store.StrategyMobile})
	fr.done = make(chan struct{})
	rec := do(t, a, http.MethodPost, "/sites/1/run", "")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rec.Code)
	}
	select {
	case <-fr.done:
	case <-time.After(2 * time.Second):
		t.Fatal("runner was not invoked")
	}
	_ = site
}
