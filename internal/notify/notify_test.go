package notify

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/t0mer/kessel/internal/crypto"
	"github.com/t0mer/kessel/internal/store"
)

func newStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "k.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func newCipher(t *testing.T) *crypto.Cipher {
	t.Helper()
	key, _ := crypto.NewKey()
	c, _ := crypto.New(key)
	return c
}

func discard() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func addShoutrrrChannel(t *testing.T, s *store.Store, c *crypto.Cipher, onSuccess, onFailure bool) store.Channel {
	t.Helper()
	cfg, _ := json.Marshal(ShoutrrrConfig{URL: "slack://tok@chan"})
	enc, _ := c.Encrypt(cfg)
	ch, err := s.CreateChannel(context.Background(), store.Channel{Type: store.ChannelShoutrrr, Name: "Slack", ConfigEncrypted: enc, NotifyOnSuccess: onSuccess, NotifyOnFailure: onFailure})
	if err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}
	return ch
}

func captureShoutrrr(t *testing.T) *[]string {
	t.Helper()
	var mu sync.Mutex
	msgs := &[]string{}
	orig := shoutrrrSend
	shoutrrrSend = func(url, message string) error {
		mu.Lock()
		*msgs = append(*msgs, message)
		mu.Unlock()
		return nil
	}
	t.Cleanup(func() { shoutrrrSend = orig })
	return msgs
}

func TestNotifyFailureSends(t *testing.T) {
	s := newStore(t)
	c := newCipher(t)
	ctx := context.Background()
	site, _ := s.CreateSite(ctx, store.Site{Name: "Ex", URL: "https://ex.com", Strategy: store.StrategyMobile})
	_, _ = s.CreateThresholdRule(ctx, store.ThresholdRule{SiteID: site.ID, Category: CategoryPerformance, Mode: store.ThresholdAbsolute, Value: 80})
	addShoutrrrChannel(t, s, c, false, true) // failure only
	msgs := captureShoutrrr(t)

	run, _ := s.CreateRun(ctx, store.Run{SiteID: site.ID, Strategy: store.StrategyMobile, Status: store.RunStatusSuccess, StartedAt: time.Unix(1000, 0), FinishedAt: time.Unix(1001, 0), Perf: f64(70)})
	n := New(s, c, discard())
	n.Notify(ctx, site, run)

	if len(*msgs) != 1 {
		t.Fatalf("sent %d messages, want 1", len(*msgs))
	}
	if got := (*msgs)[0]; !strings.Contains(got, "Ex") || !strings.Contains(got, "performance") {
		t.Errorf("message missing content: %q", got)
	}
	cnt, _ := s.CountNotificationLogs(ctx, run.ID)
	if cnt != 1 {
		t.Errorf("notification logs = %d, want 1", cnt)
	}
}

func TestNotifySuccessRespectsToggles(t *testing.T) {
	s := newStore(t)
	c := newCipher(t)
	ctx := context.Background()
	site, _ := s.CreateSite(ctx, store.Site{Name: "Ex", URL: "https://ex.com", Strategy: store.StrategyMobile})
	addShoutrrrChannel(t, s, c, false, true) // failure-only channel; run is a success -> no send
	msgs := captureShoutrrr(t)
	run, _ := s.CreateRun(ctx, store.Run{SiteID: site.ID, Strategy: store.StrategyMobile, Status: store.RunStatusSuccess, StartedAt: time.Unix(1, 0), FinishedAt: time.Unix(2, 0), Perf: f64(95)})
	New(s, c, discard()).Notify(ctx, site, run)
	if len(*msgs) != 0 {
		t.Fatalf("success run should not notify failure-only channel; sent %d", len(*msgs))
	}
}

func TestNotifySuccessSendsToSuccessChannel(t *testing.T) {
	s := newStore(t)
	c := newCipher(t)
	ctx := context.Background()
	site, _ := s.CreateSite(ctx, store.Site{Name: "Ex", URL: "https://ex.com", Strategy: store.StrategyMobile})
	addShoutrrrChannel(t, s, c, true, false) // success channel
	msgs := captureShoutrrr(t)
	run, _ := s.CreateRun(ctx, store.Run{SiteID: site.ID, Strategy: store.StrategyMobile, Status: store.RunStatusSuccess, StartedAt: time.Unix(1, 0), FinishedAt: time.Unix(2, 0), Perf: f64(95)})
	New(s, c, discard()).Notify(ctx, site, run)
	if len(*msgs) != 1 {
		t.Fatalf("success channel should receive success run; sent %d", len(*msgs))
	}
}
