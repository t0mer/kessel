package main

import (
	"bytes"
	"testing"

	"github.com/t0mer/kessel/internal/version"
)

func TestVersionFlag(t *testing.T) {
	version.Version = "2026.8.0"
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("2026.8.0")) {
		t.Fatalf("version output = %q, want it to contain the version", out.String())
	}
}

func TestServeCommandExists(t *testing.T) {
	cmd := newRootCmd()
	for _, c := range cmd.Commands() {
		if c.Name() == "serve" {
			return
		}
	}
	t.Fatal("serve subcommand not registered")
}
