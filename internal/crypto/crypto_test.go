package crypto

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func newCipher(t *testing.T) *Cipher {
	t.Helper()
	key, err := NewKey()
	if err != nil {
		t.Fatalf("NewKey: %v", err)
	}
	c, err := New(key)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	c := newCipher(t)
	pt := []byte("slack://token@channel")
	ct, err := c.Encrypt(pt)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if bytes.Equal(ct, pt) {
		t.Fatal("ciphertext equals plaintext")
	}
	back, err := c.Decrypt(ct)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(back, pt) {
		t.Fatalf("round-trip mismatch: %q", back)
	}
}

func TestEncryptNonDeterministic(t *testing.T) {
	c := newCipher(t)
	a, _ := c.Encrypt([]byte("x"))
	b, _ := c.Encrypt([]byte("x"))
	if bytes.Equal(a, b) {
		t.Fatal("two encryptions produced identical ciphertext (nonce reuse?)")
	}
}

func TestDecryptWrongKeyFails(t *testing.T) {
	c1 := newCipher(t)
	c2 := newCipher(t)
	ct, _ := c1.Encrypt([]byte("secret"))
	if _, err := c2.Decrypt(ct); err == nil {
		t.Fatal("decrypt with wrong key should fail")
	}
}

func TestDecryptTamperFails(t *testing.T) {
	c := newCipher(t)
	ct, _ := c.Encrypt([]byte("secret"))
	ct[len(ct)-1] ^= 0xff
	if _, err := c.Decrypt(ct); err == nil {
		t.Fatal("tampered ciphertext should fail GCM auth")
	}
}

func TestNewKeySizeValidation(t *testing.T) {
	if _, err := New([]byte("short")); err == nil {
		t.Fatal("expected error for short key")
	}
}

func TestParseKey(t *testing.T) {
	raw, _ := NewKey()
	s := hex.EncodeToString(raw)
	got, err := ParseKey(s)
	if err != nil || !bytes.Equal(got, raw) {
		t.Fatalf("ParseKey round-trip failed: %v", err)
	}
	if _, err := ParseKey("nothex"); err == nil {
		t.Fatal("expected error for non-hex key")
	}
	if _, err := ParseKey("abcd"); err == nil {
		t.Fatal("expected error for wrong-length key")
	}
}

func TestLoadOrCreateKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kessel.key")
	k1, err := LoadOrCreateKey(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(k1) != KeySize {
		t.Fatalf("key size = %d, want %d", len(k1), KeySize)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("key file perms = %v, want 0600", info.Mode().Perm())
	}
	k2, err := LoadOrCreateKey(path) // reuse
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !bytes.Equal(k1, k2) {
		t.Fatal("second load returned a different key")
	}
}
