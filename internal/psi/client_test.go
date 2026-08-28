package psi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func errorsAs(err error, target any) bool { return errors.As(err, target) }

func fixtureServer(t *testing.T, capture *http.Request) *httptest.Server {
	t.Helper()
	data, err := os.ReadFile("testdata/response.json")
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if capture != nil {
			*capture = *r
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}))
}

func TestRunParsesAndReturnsRaw(t *testing.T) {
	srv := fixtureServer(t, nil)
	defer srv.Close()
	c := NewClient(WithBaseURL(srv.URL))
	res, raw, err := c.Run(context.Background(), "https://example.com", "mobile")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Performance == nil || *res.Performance != 99 {
		t.Errorf("Performance = %v, want 99", res.Performance)
	}
	if len(raw) == 0 {
		t.Error("expected raw body bytes")
	}
}

func TestRunSendsQueryParams(t *testing.T) {
	var got http.Request
	srv := fixtureServer(t, &got)
	defer srv.Close()
	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("SECRET"))
	if _, _, err := c.Run(context.Background(), "https://example.com/page", "desktop"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	q := got.URL.Query()
	if q.Get("url") != "https://example.com/page" {
		t.Errorf("url param = %q", q.Get("url"))
	}
	if q.Get("strategy") != "desktop" {
		t.Errorf("strategy param = %q", q.Get("strategy"))
	}
	if q.Get("key") != "SECRET" {
		t.Errorf("key param = %q, want SECRET", q.Get("key"))
	}
	cats := q["category"]
	if len(cats) != 4 {
		t.Errorf("category params = %v, want 4", cats)
	}
}

func TestRunKeylessOmitsKey(t *testing.T) {
	var got http.Request
	srv := fixtureServer(t, &got)
	defer srv.Close()
	c := NewClient(WithBaseURL(srv.URL)) // no key
	if _, _, err := c.Run(context.Background(), "https://example.com", "mobile"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, ok := got.URL.Query()["key"]; ok {
		t.Error("key param should be absent when keyless")
	}
}

func TestNewClientDefaults(t *testing.T) {
	c := NewClient()
	if c.httpClient.Timeout != 60*time.Second {
		t.Errorf("timeout = %v, want 60s", c.httpClient.Timeout)
	}
	if c.maxRetries != 3 {
		t.Errorf("maxRetries = %d, want 3", c.maxRetries)
	}
}

func TestRunRetriesOn503(t *testing.T) {
	data, _ := os.ReadFile("testdata/response.json")
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write(data)
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithRetries(3, func(int) time.Duration { return 0 }))
	res, _, err := c.Run(context.Background(), "https://example.com", "mobile")
	if err != nil {
		t.Fatalf("Run after retries: %v", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
	if res.Performance == nil {
		t.Error("expected parsed result after successful retry")
	}
}

func TestRunFailsFastOn400(t *testing.T) {
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":400,"message":"bad url"}}`))
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithRetries(3, func(int) time.Duration { return 0 }))
	_, _, err := c.Run(context.Background(), "bad", "mobile")
	if err == nil {
		t.Fatal("expected error on 400")
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (no retry on 4xx)", attempts)
	}
	var apiErr *APIError
	if !errorsAs(err, &apiErr) || apiErr.Status != 400 {
		t.Errorf("err = %v, want *APIError with status 400", err)
	}
}

func TestRunExhaustsRetries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := NewClient(WithBaseURL(srv.URL), WithRetries(2, func(int) time.Duration { return 0 }))
	_, _, err := c.Run(context.Background(), "https://example.com", "mobile")
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
}
