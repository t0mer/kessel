package store

import (
	"context"
	"errors"
	"testing"
)

func TestCreateAndGetSite(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	created, err := s.CreateSite(ctx, Site{Name: "My Site", URL: "https://example.com", Strategy: StrategyBoth})
	if err != nil {
		t.Fatalf("CreateSite: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if created.Slug != "my-site" {
		t.Errorf("slug = %q, want my-site", created.Slug)
	}
	if !created.Enabled {
		t.Error("new site should default enabled")
	}
	if created.CreatedAt.IsZero() {
		t.Error("created_at not set")
	}

	got, err := s.GetSite(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetSite: %v", err)
	}
	if got.URL != "https://example.com" || got.Strategy != StrategyBoth {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

func TestCreateSiteUniqueSlug(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	a, _ := s.CreateSite(ctx, Site{Name: "Dup", URL: "https://a.com", Strategy: StrategyMobile})
	b, err := s.CreateSite(ctx, Site{Name: "Dup", URL: "https://b.com", Strategy: StrategyMobile})
	if err != nil {
		t.Fatalf("second CreateSite: %v", err)
	}
	if a.Slug == b.Slug {
		t.Fatalf("slugs collide: %q == %q", a.Slug, b.Slug)
	}
	if b.Slug != "dup-2" {
		t.Errorf("second slug = %q, want dup-2", b.Slug)
	}
}

func TestGetSiteNotFound(t *testing.T) {
	s := openTemp(t)
	_, err := s.GetSite(context.Background(), 999)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestUpdateSite(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	c, _ := s.CreateSite(ctx, Site{Name: "Orig", URL: "https://o.com", Strategy: StrategyMobile})
	c.Name = "Renamed"
	c.URL = "https://r.com"
	c.Strategy = StrategyDesktop
	c.Enabled = false
	upd, err := s.UpdateSite(ctx, c)
	if err != nil {
		t.Fatalf("UpdateSite: %v", err)
	}
	if upd.Name != "Renamed" || upd.URL != "https://r.com" || upd.Strategy != StrategyDesktop || upd.Enabled {
		t.Errorf("update not applied: %+v", upd)
	}
	if upd.Slug != c.Slug {
		t.Errorf("slug changed on update: %q -> %q", c.Slug, upd.Slug)
	}
}

func TestSetSiteEnabledAndList(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	c, _ := s.CreateSite(ctx, Site{Name: "Zeta", URL: "https://z.com", Strategy: StrategyMobile})
	_, _ = s.CreateSite(ctx, Site{Name: "Alpha", URL: "https://a.com", Strategy: StrategyMobile})
	if err := s.SetSiteEnabled(ctx, c.ID, false); err != nil {
		t.Fatalf("SetSiteEnabled: %v", err)
	}
	got, _ := s.GetSite(ctx, c.ID)
	if got.Enabled {
		t.Error("site should be disabled")
	}
	list, err := s.ListSites(ctx)
	if err != nil {
		t.Fatalf("ListSites: %v", err)
	}
	if len(list) != 2 || list[0].Name != "Alpha" {
		t.Errorf("list not ordered by name: %+v", list)
	}
}

func TestDeleteSiteCascades(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	c, _ := s.CreateSite(ctx, Site{Name: "Del", URL: "https://d.com", Strategy: StrategyMobile})
	_, err := s.DB().Exec(`INSERT INTO schedules(site_id, cron_expr, enabled, created_at) VALUES(?, '@every 6h', 1, 0)`, c.ID)
	if err != nil {
		t.Fatalf("insert schedule: %v", err)
	}
	if err := s.DeleteSite(ctx, c.ID); err != nil {
		t.Fatalf("DeleteSite: %v", err)
	}
	var n int
	_ = s.DB().QueryRow(`SELECT COUNT(*) FROM schedules WHERE site_id=?`, c.ID).Scan(&n)
	if n != 0 {
		t.Errorf("schedules not cascaded: %d remain", n)
	}
}
