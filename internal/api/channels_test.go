package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func bodyContains(rec *httptest.ResponseRecorder, sub string) bool {
	return strings.Contains(rec.Body.String(), sub)
}

func TestCreateChannelMaskedAndValidated(t *testing.T) {
	a, _, _, _ := newAPI(t)
	if rec := do(t, a, http.MethodPost, "/channels", `{"type":"bogus","name":"x","config":{}}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("bogus type -> %d, want 400", rec.Code)
	}
	if rec := do(t, a, http.MethodPost, "/channels", `{"type":"whatsapp_web","name":"x","config":{}}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("whatsapp_web -> %d, want 400", rec.Code)
	}
	rec := do(t, a, http.MethodPost, "/channels", `{"type":"shoutrrr","name":"Slack","config":{"url":"slack://tok@chan"},"notify_on_failure":true}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create -> %d, want 201 (body=%s)", rec.Code, rec.Body.String())
	}
	var got channelResponse
	decode(t, rec, &got)
	if got.ID == 0 || got.Type != "shoutrrr" || !got.Enabled || !got.NotifyOnFailure {
		t.Fatalf("unexpected channel: %+v", got)
	}
	list := do(t, a, http.MethodGet, "/channels", "")
	if bodyContains(list, "slack://") {
		t.Error("channel config leaked in list response")
	}
}

func TestUpdateChannelKeepsConfigWhenOmitted(t *testing.T) {
	a, _, _, _ := newAPI(t)
	do(t, a, http.MethodPost, "/channels", `{"type":"shoutrrr","name":"N","config":{"url":"slack://tok@chan"}}`)
	rec := do(t, a, http.MethodPut, "/channels/1", `{"type":"shoutrrr","name":"Renamed","notify_on_success":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("update -> %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	var got channelResponse
	decode(t, rec, &got)
	if got.Name != "Renamed" || !got.NotifyOnSuccess {
		t.Fatalf("update not applied: %+v", got)
	}
}

func TestDeleteChannel(t *testing.T) {
	a, _, _, _ := newAPI(t)
	do(t, a, http.MethodPost, "/channels", `{"type":"shoutrrr","name":"N","config":{"url":"slack://x@y"}}`)
	if rec := do(t, a, http.MethodDelete, "/channels/1", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete -> %d, want 204", rec.Code)
	}
}

func TestTestChannelGreenAPI(t *testing.T) {
	a, _, _, _ := newAPI(t)
	var hit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true }))
	defer srv.Close()
	body := `{"type":"greenapi","config":{"instance_id":"1","token":"t","phone":"9720","api_url":"` + srv.URL + `"}}`
	rec := do(t, a, http.MethodPost, "/channels/test", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("test -> %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	if !hit {
		t.Error("test-send did not call the provider")
	}
}

func TestTestChannelBadType(t *testing.T) {
	a, _, _, _ := newAPI(t)
	if rec := do(t, a, http.MethodPost, "/channels/test", `{"type":"nope","config":{}}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad type -> %d, want 400", rec.Code)
	}
}
