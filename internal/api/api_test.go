package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/t0mer/kessel/internal/crypto"
	"github.com/t0mer/kessel/internal/store"
)

type fakeRunner struct {
	mu     sync.Mutex
	called []int64
	done   chan struct{}
}

func (f *fakeRunner) RunSite(ctx context.Context, site store.Site) ([]store.Run, error) {
	f.mu.Lock()
	f.called = append(f.called, site.ID)
	f.mu.Unlock()
	if f.done != nil {
		close(f.done)
	}
	return nil, nil
}

type fakeReloader struct {
	mu    sync.Mutex
	count int
}

func (f *fakeReloader) Reload(ctx context.Context) error {
	f.mu.Lock()
	f.count++
	f.mu.Unlock()
	return nil
}

func (f *fakeReloader) reloads() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.count
}

func newAPI(t *testing.T) (*API, *store.Store, *fakeRunner, *fakeReloader) {
	t.Helper()
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "k.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	fr := &fakeRunner{}
	fl := &fakeReloader{}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	key, _ := crypto.NewKey()
	keys, _ := crypto.NewManager(key, filepath.Join(dir, "kessel.key"))
	return New(s, fr, fl, filepath.Join(dir, "reports"), filepath.Join(dir, "backups"), keys, log), s, fr, fl
}

// do performs a request against the API routes and returns the recorder.
func do(t *testing.T, a *API, method, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	rec := httptest.NewRecorder()
	a.Routes().ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(v); err != nil {
		t.Fatalf("decoding response: %v (body=%s)", err, rec.Body.String())
	}
}

func TestUnknownRoute404(t *testing.T) {
	a, _, _, _ := newAPI(t)
	rec := do(t, a, http.MethodGet, "/nope", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
