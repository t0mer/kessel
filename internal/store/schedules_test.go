package store

import (
	"context"
	"errors"
	"testing"
)

func mustSite(t *testing.T, s *Store) Site {
	t.Helper()
	site, err := s.CreateSite(context.Background(), Site{Name: "S", URL: "https://s.com", Strategy: StrategyMobile})
	if err != nil {
		t.Fatalf("CreateSite: %v", err)
	}
	return site
}

func TestCreateAndListSchedules(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	site := mustSite(t, s)
	a, err := s.CreateSchedule(ctx, Schedule{SiteID: site.ID, CronExpr: "@every 6h"})
	if err != nil {
		t.Fatalf("CreateSchedule: %v", err)
	}
	if a.ID == 0 || !a.Enabled || a.CreatedAt.IsZero() {
		t.Errorf("unexpected schedule: %+v", a)
	}
	_, _ = s.CreateSchedule(ctx, Schedule{SiteID: site.ID, CronExpr: "0 * * * *"})
	list, err := s.ListSchedulesBySite(ctx, site.ID)
	if err != nil {
		t.Fatalf("ListSchedulesBySite: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d schedules, want 2", len(list))
	}
}

func TestListEnabledSchedules(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	site := mustSite(t, s)
	on, _ := s.CreateSchedule(ctx, Schedule{SiteID: site.ID, CronExpr: "@every 1h"})
	off, _ := s.CreateSchedule(ctx, Schedule{SiteID: site.ID, CronExpr: "@every 2h"})
	if err := s.SetScheduleEnabled(ctx, off.ID, false); err != nil {
		t.Fatalf("SetScheduleEnabled: %v", err)
	}
	list, err := s.ListEnabledSchedules(ctx)
	if err != nil {
		t.Fatalf("ListEnabledSchedules: %v", err)
	}
	if len(list) != 1 || list[0].ID != on.ID {
		t.Fatalf("enabled list = %+v, want only %d", list, on.ID)
	}
}

func TestUpdateAndDeleteSchedule(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	site := mustSite(t, s)
	sc, _ := s.CreateSchedule(ctx, Schedule{SiteID: site.ID, CronExpr: "@every 6h"})
	sc.CronExpr = "@every 12h"
	sc.Enabled = false
	upd, err := s.UpdateSchedule(ctx, sc)
	if err != nil {
		t.Fatalf("UpdateSchedule: %v", err)
	}
	if upd.CronExpr != "@every 12h" || upd.Enabled {
		t.Errorf("update not applied: %+v", upd)
	}
	if err := s.DeleteSchedule(ctx, sc.ID); err != nil {
		t.Fatalf("DeleteSchedule: %v", err)
	}
	_, err = s.GetSchedule(ctx, sc.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete err = %v, want ErrNotFound", err)
	}
}
