// Package version exposes the build-injected application version.
package version

// defaultVersion is used when no version is injected via -ldflags.
const defaultVersion = "dev"

// Version is the application version, set at build time with
// -ldflags "-X github.com/t0mer/kessel/internal/version.Version=<v>".
var Version = defaultVersion

// String returns the human-readable version string.
func String() string {
	return "kessel " + Version
}
