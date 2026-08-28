package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/t0mer/kessel/internal/notify"
	"github.com/t0mer/kessel/internal/store"
)

type channelResponse struct {
	ID              int64  `json:"id"`
	Type            string `json:"type"`
	Name            string `json:"name"`
	Enabled         bool   `json:"enabled"`
	NotifyOnSuccess bool   `json:"notify_on_success"`
	NotifyOnFailure bool   `json:"notify_on_failure"`
}

type channelRequest struct {
	Type            string          `json:"type"`
	Name            string          `json:"name"`
	Enabled         *bool           `json:"enabled"`
	NotifyOnSuccess *bool           `json:"notify_on_success"`
	NotifyOnFailure *bool           `json:"notify_on_failure"`
	Config          json.RawMessage `json:"config"`
}

type testRequest struct {
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config"`
}

func toChannelResponse(c store.Channel) channelResponse {
	return channelResponse{
		ID: c.ID, Type: c.Type, Name: c.Name, Enabled: c.Enabled,
		NotifyOnSuccess: c.NotifyOnSuccess, NotifyOnFailure: c.NotifyOnFailure,
	}
}

func supportedChannelType(t string) bool {
	return t == store.ChannelShoutrrr || t == store.ChannelGreenAPI
}

func (a *API) listChannels(w http.ResponseWriter, r *http.Request) {
	chs, err := a.store.ListChannels(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]channelResponse, 0, len(chs))
	for _, c := range chs {
		out = append(out, toChannelResponse(c))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) createChannel(w http.ResponseWriter, r *http.Request) {
	var req channelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if !supportedChannelType(req.Type) {
		writeError(w, http.StatusBadRequest, "type must be shoutrrr or greenapi")
		return
	}
	if len(req.Config) == 0 {
		writeError(w, http.StatusBadRequest, "config is required")
		return
	}
	enc, err := a.keys.Cipher().Encrypt(req.Config)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "encrypting config")
		return
	}
	ch := store.Channel{
		Type: req.Type, Name: req.Name, ConfigEncrypted: enc,
		NotifyOnSuccess: boolOr(req.NotifyOnSuccess, false),
		NotifyOnFailure: boolOr(req.NotifyOnFailure, true),
	}
	created, err := a.store.CreateChannel(r.Context(), ch)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toChannelResponse(created))
}

func (a *API) loadChannel(w http.ResponseWriter, r *http.Request) (store.Channel, bool) {
	id, err := idParam(r, "channelID")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid channel id")
		return store.Channel{}, false
	}
	ch, err := a.store.GetChannel(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "channel not found")
		return store.Channel{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return store.Channel{}, false
	}
	return ch, true
}

func (a *API) updateChannel(w http.ResponseWriter, r *http.Request) {
	ch, ok := a.loadChannel(w, r)
	if !ok {
		return
	}
	var req channelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.Name) != "" {
		ch.Name = req.Name
	}
	if req.Enabled != nil {
		ch.Enabled = *req.Enabled
	}
	if req.NotifyOnSuccess != nil {
		ch.NotifyOnSuccess = *req.NotifyOnSuccess
	}
	if req.NotifyOnFailure != nil {
		ch.NotifyOnFailure = *req.NotifyOnFailure
	}
	if len(req.Config) > 0 { // re-encrypt only when a new config is supplied
		enc, err := a.keys.Cipher().Encrypt(req.Config)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "encrypting config")
			return
		}
		ch.ConfigEncrypted = enc
	}
	updated, err := a.store.UpdateChannel(r.Context(), ch)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toChannelResponse(updated))
}

func (a *API) deleteChannel(w http.ResponseWriter, r *http.Request) {
	ch, ok := a.loadChannel(w, r)
	if !ok {
		return
	}
	if err := a.store.DeleteChannel(r.Context(), ch.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) testChannel(w http.ResponseWriter, r *http.Request) {
	var req testRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	sender, err := notify.BuildSender(req.Type, req.Config, a.httpClient)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := sender.Send(r.Context(), "Kessel test notification ✅"); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func boolOr(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}
