package store

import (
	"context"
	"errors"
	"testing"
)

func TestCreateAndListThresholdRules(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	site := mustSite(t, s)
	r, err := s.CreateThresholdRule(ctx, ThresholdRule{SiteID: site.ID, Category: "performance", Mode: ThresholdAbsolute, Value: 80})
	if err != nil {
		t.Fatalf("CreateThresholdRule: %v", err)
	}
	if r.ID == 0 || !r.Enabled {
		t.Fatalf("unexpected rule: %+v", r)
	}
	_, _ = s.CreateThresholdRule(ctx, ThresholdRule{SiteID: site.ID, Category: "seo", Mode: ThresholdDelta, Value: 10})
	list, _ := s.ListThresholdRulesBySite(ctx, site.ID)
	if len(list) != 2 {
		t.Fatalf("got %d rules, want 2", len(list))
	}
}

func TestListEnabledThresholdRules(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	site := mustSite(t, s)
	on, _ := s.CreateThresholdRule(ctx, ThresholdRule{SiteID: site.ID, Category: "performance", Mode: ThresholdAbsolute, Value: 80})
	off, _ := s.CreateThresholdRule(ctx, ThresholdRule{SiteID: site.ID, Category: "seo", Mode: ThresholdAbsolute, Value: 90})
	if err := s.SetThresholdRuleEnabled(ctx, off.ID, false); err != nil {
		t.Fatalf("SetThresholdRuleEnabled: %v", err)
	}
	list, _ := s.ListEnabledThresholdRulesBySite(ctx, site.ID)
	if len(list) != 1 || list[0].ID != on.ID {
		t.Fatalf("enabled = %+v, want only %d", list, on.ID)
	}
}

func TestUpdateAndDeleteThresholdRule(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	site := mustSite(t, s)
	r, _ := s.CreateThresholdRule(ctx, ThresholdRule{SiteID: site.ID, Category: "performance", Mode: ThresholdAbsolute, Value: 80})
	r.Value = 70
	r.Mode = ThresholdDelta
	r.Enabled = false
	upd, err := s.UpdateThresholdRule(ctx, r)
	if err != nil {
		t.Fatalf("UpdateThresholdRule: %v", err)
	}
	if upd.Value != 70 || upd.Mode != ThresholdDelta || upd.Enabled {
		t.Errorf("update not applied: %+v", upd)
	}
	if err := s.DeleteThresholdRule(ctx, r.ID); err != nil {
		t.Fatalf("DeleteThresholdRule: %v", err)
	}
	if _, err := s.GetThresholdRule(ctx, r.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
