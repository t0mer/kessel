package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/t0mer/kessel/internal/store"
)

func scrape(t *testing.T, m *Metrics) string {
	t.Helper()
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("metrics status = %d", rec.Code)
	}
	return rec.Body.String()
}

func f64(v float64) *float64 { return &v }

func TestObserveRunExportsMetrics(t *testing.T) {
	m := New()
	site := store.Site{Slug: "my-site"}
	run := store.Run{Strategy: store.StrategyMobile, Status: store.RunStatusSuccess, Perf: f64(95)}
	m.ObserveRun(site, run, 3*time.Second)

	body := scrape(t, m)
	for _, want := range []string{
		"kessel_checks_total",
		"kessel_check_duration_seconds",
		"kessel_last_score",
		`site="my-site"`,
		`strategy="mobile"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics output missing %q", want)
		}
	}
}

func TestObserveNotification(t *testing.T) {
	m := New()
	m.ObserveNotification("sent")
	m.ObserveNotification("error")
	body := scrape(t, m)
	if !strings.Contains(body, "kessel_notifications_total") {
		t.Error("missing notifications counter")
	}
	if !strings.Contains(body, `status="sent"`) || !strings.Contains(body, `status="error"`) {
		t.Error("missing notification status labels")
	}
}
