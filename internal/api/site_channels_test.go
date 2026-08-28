package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/t0mer/kessel/internal/store"
)

func TestSiteChannelsSetAndGet(t *testing.T) {
	a, s, _, _ := newAPI(t)
	ctx := context.Background()
	_, _ = s.CreateSite(ctx, store.Site{Name: "S", URL: "https://s.com", Strategy: store.StrategyMobile})
	c1, _ := s.CreateChannel(ctx, store.Channel{Type: store.ChannelShoutrrr, Name: "A", ConfigEncrypted: []byte{1}})
	c2, _ := s.CreateChannel(ctx, store.Channel{Type: store.ChannelShoutrrr, Name: "B", ConfigEncrypted: []byte{2}})

	// initially empty
	rec := do(t, a, http.MethodGet, "/sites/1/channels", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get -> %d", rec.Code)
	}
	var got siteChannelsBody
	decode(t, rec, &got)
	if len(got.ChannelIDs) != 0 {
		t.Fatalf("expected no linked channels, got %v", got.ChannelIDs)
	}

	// link c1 only
	rec = do(t, a, http.MethodPut, "/sites/1/channels", `{"channel_ids":[`+itoa(c1.ID)+`]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("put -> %d (body=%s)", rec.Code, rec.Body.String())
	}
	rec = do(t, a, http.MethodGet, "/sites/1/channels", "")
	decode(t, rec, &got)
	if len(got.ChannelIDs) != 1 || got.ChannelIDs[0] != c1.ID {
		t.Fatalf("linked = %v, want [%d]", got.ChannelIDs, c1.ID)
	}
	_ = c2
}

func TestSiteChannelsRejectsUnknownChannel(t *testing.T) {
	a, s, _, _ := newAPI(t)
	_, _ = s.CreateSite(context.Background(), store.Site{Name: "S", URL: "https://s.com", Strategy: store.StrategyMobile})
	rec := do(t, a, http.MethodPut, "/sites/1/channels", `{"channel_ids":[999]}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("put unknown channel -> %d, want 400", rec.Code)
	}
}
