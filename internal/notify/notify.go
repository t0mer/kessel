package notify

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/t0mer/kessel/internal/crypto"
	"github.com/t0mer/kessel/internal/store"
)

// NotificationObserver observes notification send outcomes (metrics).
type NotificationObserver interface {
	ObserveNotification(status string)
}

// Notifier dispatches run outcomes to enabled channels.
type Notifier struct {
	store  *store.Store
	cipher *crypto.Cipher
	log    *slog.Logger
	http   *http.Client
	obs    NotificationObserver
}

// New builds a Notifier.
func New(st *store.Store, cipher *crypto.Cipher, log *slog.Logger) *Notifier {
	return &Notifier{store: st, cipher: cipher, log: log, http: &http.Client{Timeout: 20 * time.Second}}
}

// SetObserver attaches a metrics observer (optional).
func (n *Notifier) SetObserver(o NotificationObserver) { n.obs = o }

// Notify evaluates thresholds and sends to matching channels. Best-effort:
// errors are logged and recorded, never returned.
func (n *Notifier) Notify(ctx context.Context, site store.Site, run store.Run) {
	rules, err := n.store.ListEnabledThresholdRulesBySite(ctx, site.ID)
	if err != nil {
		n.log.Error("notify: loading rules", "site", site.Slug, "error", err)
		return
	}
	var prev *store.Run
	if p, err := n.store.PreviousSuccessfulRun(ctx, site.ID, run.Strategy, run.StartedAt); err == nil {
		prev = &p
	}
	breaches := Evaluate(rules, run, prev)
	failure := run.Status == store.RunStatusError || len(breaches) > 0

	channels, err := n.store.ListEnabledChannels(ctx)
	if err != nil {
		n.log.Error("notify: loading channels", "error", err)
		return
	}
	if len(channels) == 0 {
		return
	}
	message := BuildMessage(site, run, breaches)

	for _, ch := range channels {
		if !((failure && ch.NotifyOnFailure) || (!failure && ch.NotifyOnSuccess)) {
			continue
		}
		n.sendTo(ctx, ch, run, message)
	}
}

func (n *Notifier) sendTo(ctx context.Context, ch store.Channel, run store.Run, message string) {
	status, errStr := "sent", ""
	cfg, err := n.cipher.Decrypt(ch.ConfigEncrypted)
	if err != nil {
		status, errStr = "error", "decrypt config: "+err.Error()
	} else if sender, berr := BuildSender(ch.Type, cfg, n.http); berr != nil {
		status, errStr = "error", berr.Error()
	} else if serr := sender.Send(ctx, message); serr != nil {
		status, errStr = "error", serr.Error()
	}
	if status == "error" {
		n.log.Error("notify: send failed", "channel", ch.Name, "error", errStr)
	}
	if n.obs != nil {
		n.obs.ObserveNotification(status)
	}
	if lerr := n.store.LogNotification(ctx, store.NotificationLog{RunID: run.ID, ChannelID: ch.ID, Status: status, Error: errStr}); lerr != nil {
		n.log.Error("notify: logging", "error", lerr)
	}
}

// BuildMessage renders the notification text.
func BuildMessage(site store.Site, run store.Run, breaches []Breach) string {
	var b strings.Builder
	switch {
	case run.Status == store.RunStatusError:
		fmt.Fprintf(&b, "Kessel ⚠️ %s (%s): check FAILED\n%s\n", site.Name, run.Strategy, run.Error)
	case len(breaches) > 0:
		fmt.Fprintf(&b, "Kessel ⚠️ %s (%s): %d threshold breach(es)\n", site.Name, run.Strategy, len(breaches))
		for _, br := range breaches {
			if br.Rule.Mode == store.ThresholdDelta && br.Previous != nil && br.Current != nil {
				fmt.Fprintf(&b, "• %s dropped %.0f→%.0f (limit -%.0f)\n", br.Rule.Category, *br.Previous, *br.Current, br.Rule.Value)
			} else if br.Current != nil {
				fmt.Fprintf(&b, "• %s %.0f below %.0f\n", br.Rule.Category, *br.Current, br.Rule.Value)
			}
		}
	default:
		fmt.Fprintf(&b, "Kessel ✅ %s (%s): all thresholds OK\n", site.Name, run.Strategy)
	}
	fmt.Fprintf(&b, "%s\n", site.URL)
	if run.ReportPath != "" {
		fmt.Fprintf(&b, "Report: %s\n", run.ReportPath)
	}
	return b.String()
}
