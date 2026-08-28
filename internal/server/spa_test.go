package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func testDist() fstest.MapFS {
	return fstest.MapFS{
		"index.html":    {Data: []byte("<html>INDEX</html>")},
		"assets/app.js": {Data: []byte("APPJS")},
	}
}

func TestSPAServesAsset(t *testing.T) {
	h := SPAHandler(testDist())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "APPJS" {
		t.Fatalf("asset: code=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestSPAFallsBackToIndex(t *testing.T) {
	h := SPAHandler(testDist())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sites/42", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "<html>INDEX</html>" {
		t.Fatalf("fallback: code=%d body=%q", rec.Code, rec.Body.String())
	}
}
