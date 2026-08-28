package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/t0mer/kessel/internal/store"
)

func TestCreateScheduleValidatesCron(t *testing.T) {
	a, s, _, _ := newAPI(t)
	_, _ = s.CreateSite(context.Background(), store.Site{Name: "S", URL: "https://s.com", Strategy: store.StrategyMobile})
	rec := do(t, a, http.MethodPost, "/sites/1/schedules", `{"cron_expr":"nonsense"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCreateScheduleReloads(t *testing.T) {
	a, s, _, fl := newAPI(t)
	_, _ = s.CreateSite(context.Background(), store.Site{Name: "S", URL: "https://s.com", Strategy: store.StrategyMobile})
	rec := do(t, a, http.MethodPost, "/sites/1/schedules", `{"cron_expr":"@every 6h"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body=%s)", rec.Code, rec.Body.String())
	}
	var sched scheduleResponse
	decode(t, rec, &sched)
	if sched.ID == 0 || sched.CronExpr != "@every 6h" || !sched.Enabled {
		t.Fatalf("unexpected schedule: %+v", sched)
	}
	if fl.reloads() != 1 {
		t.Fatalf("reloads = %d, want 1", fl.reloads())
	}
}

func TestListSchedules(t *testing.T) {
	a, s, _, _ := newAPI(t)
	site, _ := s.CreateSite(context.Background(), store.Site{Name: "S", URL: "https://s.com", Strategy: store.StrategyMobile})
	_, _ = s.CreateSchedule(context.Background(), store.Schedule{SiteID: site.ID, CronExpr: "@every 6h"})
	rec := do(t, a, http.MethodGet, "/sites/1/schedules", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var list []scheduleResponse
	decode(t, rec, &list)
	if len(list) != 1 {
		t.Fatalf("got %d, want 1", len(list))
	}
}

func TestUpdateAndDeleteSchedule(t *testing.T) {
	a, s, _, fl := newAPI(t)
	site, _ := s.CreateSite(context.Background(), store.Site{Name: "S", URL: "https://s.com", Strategy: store.StrategyMobile})
	sc, _ := s.CreateSchedule(context.Background(), store.Schedule{SiteID: site.ID, CronExpr: "@every 6h"})

	rec := do(t, a, http.MethodPut, "/schedules/1", `{"cron_expr":"@every 12h","enabled":false}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	var upd scheduleResponse
	decode(t, rec, &upd)
	if upd.CronExpr != "@every 12h" || upd.Enabled {
		t.Fatalf("update not applied: %+v", upd)
	}

	rec = do(t, a, http.MethodDelete, "/schedules/1", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", rec.Code)
	}
	if fl.reloads() < 2 {
		t.Fatalf("reloads = %d, want >= 2 (update + delete)", fl.reloads())
	}
	_ = sc
}
