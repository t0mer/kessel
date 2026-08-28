package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/t0mer/kessel/internal/store"
)

type siteChannelsBody struct {
	ChannelIDs []int64 `json:"channel_ids"`
}

func (a *API) getSiteChannels(w http.ResponseWriter, r *http.Request) {
	site, ok := a.loadSite(w, r)
	if !ok {
		return
	}
	ids, err := a.store.ListChannelIDsBySite(r.Context(), site.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if ids == nil {
		ids = []int64{}
	}
	writeJSON(w, http.StatusOK, siteChannelsBody{ChannelIDs: ids})
}

func (a *API) putSiteChannels(w http.ResponseWriter, r *http.Request) {
	site, ok := a.loadSite(w, r)
	if !ok {
		return
	}
	var req siteChannelsBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	for _, id := range req.ChannelIDs {
		if _, err := a.store.GetChannel(r.Context(), id); errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusBadRequest, "unknown channel id")
			return
		} else if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if err := a.store.SetSiteChannels(r.Context(), site.ID, req.ChannelIDs); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	ids := req.ChannelIDs
	if ids == nil {
		ids = []int64{}
	}
	writeJSON(w, http.StatusOK, siteChannelsBody{ChannelIDs: ids})
}
