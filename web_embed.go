// Package webembed embeds the built frontend SPA (web/dist) for serving.
package webembed

import (
	"embed"
	"io/fs"
)

//go:embed web/dist
var distFiles embed.FS

// FS returns the web/dist subtree as an fs.FS rooted at dist.
func FS() (fs.FS, error) {
	return fs.Sub(distFiles, "web/dist")
}
