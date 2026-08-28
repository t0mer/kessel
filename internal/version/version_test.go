package version

import (
	"strings"
	"testing"
)

func TestStringContainsVersion(t *testing.T) {
	Version = "2026.8.0"
	got := String()
	if !strings.Contains(got, "2026.8.0") {
		t.Fatalf("String() = %q, want it to contain the version", got)
	}
}

func TestDefaultVersionIsDev(t *testing.T) {
	// New builds without ldflags default to "dev".
	if defaultVersion != "dev" {
		t.Fatalf("defaultVersion = %q, want dev", defaultVersion)
	}
}
