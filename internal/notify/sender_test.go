package notify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/t0mer/kessel/internal/store"
)

func TestBuildSenderShoutrrr(t *testing.T) {
	cfg, _ := json.Marshal(ShoutrrrConfig{URL: "slack://tok@chan"})
	var gotURL, gotMsg string
	orig := shoutrrrSend
	shoutrrrSend = func(url, message string) error { gotURL, gotMsg = url, message; return nil }
	defer func() { shoutrrrSend = orig }()

	s, err := BuildSender(store.ChannelShoutrrr, cfg, http.DefaultClient)
	if err != nil {
		t.Fatalf("BuildSender: %v", err)
	}
	if err := s.Send(context.Background(), "hello"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gotURL != "slack://tok@chan" || gotMsg != "hello" {
		t.Errorf("shoutrrr got url=%q msg=%q", gotURL, gotMsg)
	}
}

func TestBuildSenderGreenAPI(t *testing.T) {
	var body map[string]string
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg, _ := json.Marshal(GreenAPIConfig{InstanceID: " 123 ", Token: " tok ", Phone: " 972501234567 ", APIURL: srv.URL})
	s, err := BuildSender(store.ChannelGreenAPI, cfg, srv.Client())
	if err != nil {
		t.Fatalf("BuildSender: %v", err)
	}
	if err := s.Send(context.Background(), "hi"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gotPath != "/waInstance123/sendMessage/tok" {
		t.Errorf("path = %q, want trimmed /waInstance123/sendMessage/tok", gotPath)
	}
	if body["chatId"] != "972501234567@c.us" || body["message"] != "hi" {
		t.Errorf("body = %+v", body)
	}
}

func TestGreenAPISendErrorOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad"))
	}))
	defer srv.Close()
	cfg, _ := json.Marshal(GreenAPIConfig{InstanceID: "1", Token: "t", Phone: "9720000@c.us", APIURL: srv.URL})
	s, _ := BuildSender(store.ChannelGreenAPI, cfg, srv.Client())
	if err := s.Send(context.Background(), "x"); err == nil {
		t.Fatal("expected error on 400")
	}
}

func TestBuildSenderUnsupported(t *testing.T) {
	if _, err := BuildSender(store.ChannelWhatsAppWeb, []byte(`{}`), http.DefaultClient); err == nil {
		t.Fatal("whatsapp_web should be unsupported for now")
	}
	if _, err := BuildSender("bogus", []byte(`{}`), http.DefaultClient); err == nil {
		t.Fatal("unknown type should error")
	}
}

func TestGreenAPIPhoneAlreadyHasSuffix(t *testing.T) {
	var body map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
	}))
	defer srv.Close()
	cfg, _ := json.Marshal(GreenAPIConfig{InstanceID: "1", Token: "t", Phone: "9720000@c.us", APIURL: srv.URL})
	s, _ := BuildSender(store.ChannelGreenAPI, cfg, srv.Client())
	_ = s.Send(context.Background(), "x")
	if strings.Count(body["chatId"], "@c.us") != 1 {
		t.Errorf("chatId double-suffixed: %q", body["chatId"])
	}
}
