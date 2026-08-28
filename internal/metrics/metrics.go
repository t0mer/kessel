// Package metrics exposes Prometheus collectors for Kessel.
package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/t0mer/kessel/internal/notify"
	"github.com/t0mer/kessel/internal/store"
)

// Metrics holds Kessel's Prometheus collectors on a private registry.
type Metrics struct {
	reg           *prometheus.Registry
	checksTotal   *prometheus.CounterVec
	checkDuration *prometheus.HistogramVec
	notifications *prometheus.CounterVec
	lastScore     *prometheus.GaugeVec
}

// New builds and registers the collectors.
func New() *Metrics {
	reg := prometheus.NewRegistry()
	m := &Metrics{
		reg: reg,
		checksTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "kessel_checks_total", Help: "PSI checks run, by site, strategy, and status.",
		}, []string{"site", "strategy", "status"}),
		checkDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "kessel_check_duration_seconds", Help: "PSI check duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"site", "strategy"}),
		notifications: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "kessel_notifications_total", Help: "Notifications sent, by status.",
		}, []string{"status"}),
		lastScore: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "kessel_last_score", Help: "Latest category score (0-100) per site and strategy.",
		}, []string{"site", "strategy", "category"}),
	}
	reg.MustRegister(m.checksTotal, m.checkDuration, m.notifications, m.lastScore)
	return m
}

// Handler serves the metrics in Prometheus text format.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{})
}

// ObserveRun records a completed run.
func (m *Metrics) ObserveRun(site store.Site, run store.Run, dur time.Duration) {
	m.checksTotal.WithLabelValues(site.Slug, run.Strategy, run.Status).Inc()
	m.checkDuration.WithLabelValues(site.Slug, run.Strategy).Observe(dur.Seconds())
	m.setScore(site.Slug, run.Strategy, notify.CategoryPerformance, run.Perf)
	m.setScore(site.Slug, run.Strategy, notify.CategoryAccessibility, run.Accessibility)
	m.setScore(site.Slug, run.Strategy, notify.CategoryBestPractices, run.BestPractices)
	m.setScore(site.Slug, run.Strategy, notify.CategorySEO, run.SEO)
}

func (m *Metrics) setScore(slug, strategy, category string, v *float64) {
	if v == nil {
		return
	}
	m.lastScore.WithLabelValues(slug, strategy, category).Set(*v)
}

// ObserveNotification records a notification send outcome.
func (m *Metrics) ObserveNotification(status string) {
	m.notifications.WithLabelValues(status).Inc()
}
