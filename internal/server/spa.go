package server

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// SPAHandler serves static files from dist, falling back to index.html for
// paths that do not resolve to a file (client-side routing).
func SPAHandler(dist fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(dist, p); err != nil {
			// Not a real file: serve the SPA entrypoint.
			serveIndex(w, r, dist)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

func serveIndex(w http.ResponseWriter, r *http.Request, dist fs.FS) {
	data, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		http.Error(w, "index.html not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

// MountSPA registers the SPA handler as the catch-all route.
func (s *Server) MountSPA(dist fs.FS) {
	s.mux.NotFound(SPAHandler(dist).ServeHTTP)
}
