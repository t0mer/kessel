package store

import (
	"context"
	"errors"
	"testing"
)

func TestCreateAndGetChannel(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	in := Channel{Type: ChannelShoutrrr, Name: "Slack", ConfigEncrypted: []byte{1, 2, 3}, NotifyOnFailure: true}
	c, err := s.CreateChannel(ctx, in)
	if err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}
	if c.ID == 0 || !c.Enabled {
		t.Fatalf("unexpected channel: %+v", c)
	}
	got, err := s.GetChannel(ctx, c.ID)
	if err != nil {
		t.Fatalf("GetChannel: %v", err)
	}
	if got.Type != ChannelShoutrrr || got.Name != "Slack" || len(got.ConfigEncrypted) != 3 {
		t.Errorf("mismatch: %+v", got)
	}
	if !got.NotifyOnFailure || got.NotifyOnSuccess {
		t.Errorf("notify flags wrong: %+v", got)
	}
}

func TestListChannelsAndEnabled(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	a, _ := s.CreateChannel(ctx, Channel{Type: ChannelShoutrrr, Name: "Bravo", ConfigEncrypted: []byte{1}})
	_, _ = s.CreateChannel(ctx, Channel{Type: ChannelGreenAPI, Name: "Alpha", ConfigEncrypted: []byte{2}})
	if err := s.SetChannelEnabled(ctx, a.ID, false); err != nil {
		t.Fatalf("SetChannelEnabled: %v", err)
	}
	all, _ := s.ListChannels(ctx)
	if len(all) != 2 || all[0].Name != "Alpha" {
		t.Errorf("list not ordered by name: %+v", all)
	}
	enabled, _ := s.ListEnabledChannels(ctx)
	if len(enabled) != 1 || enabled[0].Name != "Alpha" {
		t.Errorf("enabled = %+v, want only Alpha", enabled)
	}
}

func TestUpdateChannel(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	c, _ := s.CreateChannel(ctx, Channel{Type: ChannelShoutrrr, Name: "N", ConfigEncrypted: []byte{1}})
	c.Name = "Renamed"
	c.ConfigEncrypted = []byte{9, 9}
	c.NotifyOnSuccess = true
	c.Enabled = false
	upd, err := s.UpdateChannel(ctx, c)
	if err != nil {
		t.Fatalf("UpdateChannel: %v", err)
	}
	if upd.Name != "Renamed" || len(upd.ConfigEncrypted) != 2 || !upd.NotifyOnSuccess || upd.Enabled {
		t.Errorf("update not applied: %+v", upd)
	}
}

func TestDeleteChannel(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	c, _ := s.CreateChannel(ctx, Channel{Type: ChannelShoutrrr, Name: "N", ConfigEncrypted: []byte{1}})
	if err := s.DeleteChannel(ctx, c.ID); err != nil {
		t.Fatalf("DeleteChannel: %v", err)
	}
	if _, err := s.GetChannel(ctx, c.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
