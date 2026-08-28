package config

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestDefaults(t *testing.T) {
	d := Defaults()
	if d.Port != 8080 {
		t.Errorf("Port = %d, want 8080", d.Port)
	}
	if d.DataDir != "/data" {
		t.Errorf("DataDir = %q, want /data", d.DataDir)
	}
	if d.PSIConcurrency != 2 {
		t.Errorf("PSIConcurrency = %d, want 2", d.PSIConcurrency)
	}
	if d.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", d.LogLevel)
	}
	if d.LogFormat != "json" {
		t.Errorf("LogFormat = %q, want json", d.LogFormat)
	}
}

func TestValidateRejectsBadPort(t *testing.T) {
	c := Defaults()
	c.Port = 0
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for port 0")
	}
}

func TestValidateRejectsBadLogLevel(t *testing.T) {
	c := Defaults()
	c.LogLevel = "loud"
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for invalid log level")
	}
}

func TestValidateRejectsBadConcurrency(t *testing.T) {
	c := Defaults()
	c.PSIConcurrency = 0
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for concurrency < 1")
	}
}

func TestValidateAcceptsDefaults(t *testing.T) {
	if err := Defaults().Validate(); err != nil {
		t.Fatalf("defaults should validate, got %v", err)
	}
}

func TestLoadEnvOverridesDefault(t *testing.T) {
	t.Setenv("KESSEL_PORT", "9999")
	cmd := &cobra.Command{}
	RegisterFlags(cmd)
	c, err := Load(cmd)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Port != 9999 {
		t.Fatalf("Port = %d, want 9999 (env override)", c.Port)
	}
}

func TestLoadFlagOverridesEnv(t *testing.T) {
	t.Setenv("KESSEL_PORT", "9999")
	cmd := &cobra.Command{}
	RegisterFlags(cmd)
	if err := cmd.Flags().Set("port", "7777"); err != nil {
		t.Fatal(err)
	}
	c, err := Load(cmd)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Port != 7777 {
		t.Fatalf("Port = %d, want 7777 (flag beats env)", c.Port)
	}
}
