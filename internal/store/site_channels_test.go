package store

import (
	"context"
	"testing"
)

func TestSetAndListSiteChannels(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	site := mustSite(t, s)
	c1, _ := s.CreateChannel(ctx, Channel{Type: ChannelShoutrrr, Name: "A", ConfigEncrypted: []byte{1}})
	c2, _ := s.CreateChannel(ctx, Channel{Type: ChannelShoutrrr, Name: "B", ConfigEncrypted: []byte{2}})
	c3, _ := s.CreateChannel(ctx, Channel{Type: ChannelShoutrrr, Name: "C", ConfigEncrypted: []byte{3}})

	if err := s.SetSiteChannels(ctx, site.ID, []int64{c1.ID, c3.ID}); err != nil {
		t.Fatalf("SetSiteChannels: %v", err)
	}
	ids, err := s.ListChannelIDsBySite(ctx, site.ID)
	if err != nil {
		t.Fatalf("ListChannelIDsBySite: %v", err)
	}
	if len(ids) != 2 || !contains(ids, c1.ID) || !contains(ids, c3.ID) || contains(ids, c2.ID) {
		t.Fatalf("linked ids = %v, want [%d %d]", ids, c1.ID, c3.ID)
	}

	// Replace the set.
	if err := s.SetSiteChannels(ctx, site.ID, []int64{c2.ID}); err != nil {
		t.Fatalf("SetSiteChannels replace: %v", err)
	}
	ids, _ = s.ListChannelIDsBySite(ctx, site.ID)
	if len(ids) != 1 || ids[0] != c2.ID {
		t.Fatalf("after replace ids = %v, want [%d]", ids, c2.ID)
	}
}

func TestListEnabledChannelsBySite(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	site := mustSite(t, s)
	other := mustSiteNamed(t, s, "Other")
	c1, _ := s.CreateChannel(ctx, Channel{Type: ChannelShoutrrr, Name: "Linked", ConfigEncrypted: []byte{1}})
	c2, _ := s.CreateChannel(ctx, Channel{Type: ChannelShoutrrr, Name: "Disabled", ConfigEncrypted: []byte{2}})
	c3, _ := s.CreateChannel(ctx, Channel{Type: ChannelShoutrrr, Name: "OtherSite", ConfigEncrypted: []byte{3}})

	if err := s.SetSiteChannels(ctx, site.ID, []int64{c1.ID, c2.ID}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetSiteChannels(ctx, other.ID, []int64{c3.ID}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetChannelEnabled(ctx, c2.ID, false); err != nil {
		t.Fatal(err)
	}

	got, err := s.ListEnabledChannelsBySite(ctx, site.ID)
	if err != nil {
		t.Fatalf("ListEnabledChannelsBySite: %v", err)
	}
	if len(got) != 1 || got[0].ID != c1.ID {
		t.Fatalf("enabled linked = %+v, want only %d (Linked, enabled)", got, c1.ID)
	}
}

func TestDeletingChannelUnlinks(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	site := mustSite(t, s)
	c1, _ := s.CreateChannel(ctx, Channel{Type: ChannelShoutrrr, Name: "A", ConfigEncrypted: []byte{1}})
	_ = s.SetSiteChannels(ctx, site.ID, []int64{c1.ID})
	if err := s.DeleteChannel(ctx, c1.ID); err != nil {
		t.Fatal(err)
	}
	ids, _ := s.ListChannelIDsBySite(ctx, site.ID)
	if len(ids) != 0 {
		t.Fatalf("link should cascade on channel delete, got %v", ids)
	}
}

func contains(xs []int64, v int64) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func mustSiteNamed(t *testing.T, s *Store, name string) Site {
	t.Helper()
	site, err := s.CreateSite(context.Background(), Site{Name: name, URL: "https://" + name + ".com", Strategy: StrategyMobile})
	if err != nil {
		t.Fatalf("CreateSite: %v", err)
	}
	return site
}
