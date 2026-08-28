// Package server wires the chi router and HTTP lifecycle for Kessel.
package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server owns the HTTP router and underlying http.Server.
type Server struct {
	log  *slog.Logger
	http *http.Server
	mux  *chi.Mux
}

// New builds a Server listening on addr (host:port).
func New(log *slog.Logger, addr string) *Server {
	s := &Server{log: log}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/healthz", s.handleHealthz)
	r.Route("/api/v1", func(r chi.Router) {
		// endpoints mounted in later phases
	})

	s.mux = r
	s.http = &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return s
}

// Router returns the HTTP handler (useful for tests).
func (s *Server) Router() http.Handler { return s.mux }

// Start begins serving and blocks until the server stops.
func (s *Server) Start() error {
	s.log.Info("http server starting", "addr", s.http.Addr)
	if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}
