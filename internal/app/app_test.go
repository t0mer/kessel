package app

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/t0mer/kessel/internal/config"
)

func testDist() fstest.MapFS {
	return fstest.MapFS{"index.html": {Data: []byte("<html>SPA</html>")}}
}

func newTestApp(t *testing.T) *App {
	t.Helper()
	cfg := config.Defaults()
	cfg.DataDir = t.TempDir()
	cfg.LogFormat = "text"
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	a, err := New(cfg, log, testDist())
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = a.Shutdown(ctx)
	})
	return a
}

func req(t *testing.T, a *App, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	a.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestAppServesHealthz(t *testing.T) {
	a := newTestApp(t)
	if rec := req(t, a, "/healthz"); rec.Code != http.StatusOK {
		t.Fatalf("/healthz = %d, want 200", rec.Code)
	}
}

func TestAppServesAPI(t *testing.T) {
	a := newTestApp(t)
	rec := req(t, a, "/api/v1/sites")
	if rec.Code != http.StatusOK {
		t.Fatalf("/api/v1/sites = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "[]\n" {
		t.Errorf("empty sites body = %q, want []", rec.Body.String())
	}
}

func TestAppServesSPA(t *testing.T) {
	a := newTestApp(t)
	rec := req(t, a, "/dashboard")
	if rec.Code != http.StatusOK || rec.Body.String() != "<html>SPA</html>" {
		t.Fatalf("SPA fallback: code=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestAppStartSchedulerNoSchedules(t *testing.T) {
	a := newTestApp(t)
	if err := a.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
}

func TestNewCreatesDataDirAndDB(t *testing.T) {
	a := newTestApp(t)
	rec := httptest.NewRecorder()
	body := `{"name":"X","url":"https://x.com","strategy":"mobile"}`
	a.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/sites", strings.NewReader(body)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create site = %d, want 201 (body=%s)", rec.Code, rec.Body.String())
	}
}
