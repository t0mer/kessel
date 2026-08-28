package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMountMetrics(t *testing.T) {
	srv := newTestServer()
	srv.MountMetrics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("# metrics"))
	}))
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "# metrics" {
		t.Fatalf("metrics mount: code=%d body=%q", rec.Code, rec.Body.String())
	}
}
