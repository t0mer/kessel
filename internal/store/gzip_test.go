package store

import (
	"bytes"
	"testing"
)

func TestGzipRoundTrip(t *testing.T) {
	orig := []byte(`{"hello":"world","n":123}`)
	gz, err := GzipBytes(orig)
	if err != nil {
		t.Fatalf("GzipBytes: %v", err)
	}
	if len(gz) == 0 {
		t.Fatal("gzip output empty")
	}
	back, err := GunzipBytes(gz)
	if err != nil {
		t.Fatalf("GunzipBytes: %v", err)
	}
	if !bytes.Equal(back, orig) {
		t.Fatalf("round-trip mismatch: %q != %q", back, orig)
	}
}

func TestGunzipInvalid(t *testing.T) {
	if _, err := GunzipBytes([]byte("not gzip")); err == nil {
		t.Fatal("expected error for invalid gzip data")
	}
}
