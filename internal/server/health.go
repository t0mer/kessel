package server

import (
	"encoding/json"
	"net/http"

	"github.com/t0mer/kessel/internal/version"
)

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"version": version.String(),
	})
}
