package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/containrrr/shoutrrr"

	"github.com/t0mer/kessel/internal/store"
)

// Sender delivers a message over one channel.
type Sender interface {
	Send(ctx context.Context, message string) error
}

// ShoutrrrConfig is the config for a Shoutrrr channel.
type ShoutrrrConfig struct {
	URL string `json:"url"`
}

// GreenAPIConfig is the config for a GreenAPI (WhatsApp cloud) channel.
type GreenAPIConfig struct {
	InstanceID string `json:"instance_id"`
	Token      string `json:"token"`
	Phone      string `json:"phone"`
	APIURL     string `json:"api_url"`
}

// shoutrrrSend is overridable in tests.
var shoutrrrSend = func(url, message string) error { return shoutrrr.Send(url, message) }

// BuildSender constructs a Sender from a channel type and its decrypted config.
func BuildSender(channelType string, configJSON []byte, httpClient *http.Client) (Sender, error) {
	switch channelType {
	case store.ChannelShoutrrr:
		var c ShoutrrrConfig
		if err := json.Unmarshal(configJSON, &c); err != nil {
			return nil, fmt.Errorf("parsing shoutrrr config: %w", err)
		}
		if strings.TrimSpace(c.URL) == "" {
			return nil, fmt.Errorf("shoutrrr url is required")
		}
		return &shoutrrrSender{url: strings.TrimSpace(c.URL)}, nil
	case store.ChannelGreenAPI:
		var c GreenAPIConfig
		if err := json.Unmarshal(configJSON, &c); err != nil {
			return nil, fmt.Errorf("parsing greenapi config: %w", err)
		}
		return &greenapiSender{cfg: c, http: httpClient}, nil
	case store.ChannelWhatsAppWeb:
		return nil, fmt.Errorf("whatsapp_web channel not yet supported")
	default:
		return nil, fmt.Errorf("unknown channel type %q", channelType)
	}
}

type shoutrrrSender struct{ url string }

func (s *shoutrrrSender) Send(_ context.Context, message string) error {
	if err := shoutrrrSend(s.url, message); err != nil {
		return fmt.Errorf("shoutrrr send: %w", err)
	}
	return nil
}

type greenapiSender struct {
	cfg  GreenAPIConfig
	http *http.Client
}

func (g *greenapiSender) Send(ctx context.Context, message string) error {
	instanceID := strings.TrimSpace(g.cfg.InstanceID)
	token := strings.TrimSpace(g.cfg.Token)
	phone := strings.TrimSpace(g.cfg.Phone)
	apiURL := strings.TrimSpace(g.cfg.APIURL)
	if apiURL == "" {
		apiURL = "https://api.green-api.com"
	}
	if instanceID == "" || token == "" || phone == "" {
		return fmt.Errorf("greenapi requires instance_id, token, and phone")
	}
	chatID := phone
	if !strings.Contains(chatID, "@") {
		chatID += "@c.us"
	}
	endpoint := fmt.Sprintf("%s/waInstance%s/sendMessage/%s", strings.TrimRight(apiURL, "/"), instanceID, token)
	payload, _ := json.Marshal(map[string]string{"chatId": chatID, "message": message})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("building greenapi request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.http.Do(req)
	if err != nil {
		return fmt.Errorf("greenapi request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 500))
		return fmt.Errorf("greenapi returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
