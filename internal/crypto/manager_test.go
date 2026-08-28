package crypto

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestSetKeyChangesCipher(t *testing.T) {
	k1, _ := NewKey()
	k2, _ := NewKey()
	c, _ := New(k1)
	ct, _ := c.Encrypt([]byte("secret"))

	if err := c.SetKey(k2); err != nil {
		t.Fatalf("SetKey: %v", err)
	}
	// Old ciphertext no longer decrypts under the new key.
	if _, err := c.Decrypt(ct); err == nil {
		t.Fatal("expected decrypt to fail after key change")
	}
	// New round-trip works.
	ct2, _ := c.Encrypt([]byte("hi"))
	pt, err := c.Decrypt(ct2)
	if err != nil || string(pt) != "hi" {
		t.Fatalf("round-trip after SetKey failed: %v %q", err, pt)
	}
}

func TestManagerAdoptPersistsAndReKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kessel.key")
	k1, _ := NewKey()
	if err := os.WriteFile(path, []byte(hex.EncodeToString(k1)), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := NewManager(k1, path)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if m.KeyHex() != hex.EncodeToString(k1) {
		t.Fatalf("KeyHex mismatch")
	}

	k2, _ := NewKey()
	// Something encrypted with k2 (as if from another install).
	c2, _ := New(k2)
	ct, _ := c2.Encrypt([]byte("from elsewhere"))

	if err := m.Adopt(k2); err != nil {
		t.Fatalf("Adopt: %v", err)
	}
	// The manager's cipher now decrypts k2 data.
	pt, err := m.Cipher().Decrypt(ct)
	if err != nil || string(pt) != "from elsewhere" {
		t.Fatalf("after Adopt, decrypt failed: %v %q", err, pt)
	}
	// KeyHex reflects k2 and the file was updated.
	if m.KeyHex() != hex.EncodeToString(k2) {
		t.Fatalf("KeyHex not updated after Adopt")
	}
	onDisk, _ := os.ReadFile(path)
	if !bytes.Equal(onDisk, []byte(hex.EncodeToString(k2))) {
		t.Fatalf("key file not persisted with adopted key")
	}
}

func TestManagerAdoptInvalidKeyLeavesStateUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kessel.key")
	k1, _ := NewKey()
	_ = os.WriteFile(path, []byte(hex.EncodeToString(k1)), 0o600)
	m, _ := NewManager(k1, path)
	ct, _ := m.Cipher().Encrypt([]byte("x"))

	if err := m.Adopt([]byte("too-short")); err == nil {
		t.Fatal("expected error adopting an invalid key")
	}
	// Cipher, in-memory key, and file are all still the original key.
	if pt, err := m.Cipher().Decrypt(ct); err != nil || string(pt) != "x" {
		t.Fatal("cipher changed despite failed Adopt")
	}
	if m.KeyHex() != hex.EncodeToString(k1) {
		t.Fatal("in-memory key changed despite failed Adopt")
	}
	onDisk, _ := os.ReadFile(path)
	if string(onDisk) != hex.EncodeToString(k1) {
		t.Fatal("key file changed despite failed Adopt")
	}
}

func TestManagerAdoptNoFileWhenEnvKey(t *testing.T) {
	k1, _ := NewKey()
	m, _ := NewManager(k1, "") // env/flag-provided key: no file
	if m.PersistsToFile() {
		t.Fatal("expected PersistsToFile false for env key")
	}
	k2, _ := NewKey()
	if err := m.Adopt(k2); err != nil {
		t.Fatalf("Adopt with no path should still re-key: %v", err)
	}
	if m.KeyHex() != hex.EncodeToString(k2) {
		t.Fatal("KeyHex not updated")
	}
}
