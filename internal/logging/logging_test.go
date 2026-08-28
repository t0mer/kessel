package logging

import (
	"strings"
	"testing"
)

func TestNewJSONLoggerEmitsJSON(t *testing.T) {
	var sb strings.Builder
	l := newWithWriter(&sb, "info", "json")
	l.Info("hello", "k", "v")
	out := sb.String()
	if !strings.Contains(out, `"msg":"hello"`) || !strings.Contains(out, `"k":"v"`) {
		t.Fatalf("expected JSON output, got %q", out)
	}
}

func TestNewTextLoggerEmitsText(t *testing.T) {
	var sb strings.Builder
	l := newWithWriter(&sb, "debug", "text")
	l.Debug("hi")
	if !strings.Contains(sb.String(), "hi") {
		t.Fatalf("expected text output, got %q", sb.String())
	}
}

func TestUnknownLevelFallsBackToInfo(t *testing.T) {
	if levelFromString("nonsense").Level() != levelFromString("info").Level() {
		t.Fatal("unknown level should fall back to info")
	}
}
