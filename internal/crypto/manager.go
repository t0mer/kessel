package crypto

import (
	"encoding/hex"
	"fmt"
	"os"
	"sync"
)

// Manager owns the active encryption key, the Cipher built from it, and where
// the key is persisted. It lets a restore adopt a key carried in a backup
// archive: the live Cipher is re-keyed and (if backed by a file) the new key is
// written to disk so it survives a restart.
type Manager struct {
	cipher *Cipher

	mu   sync.Mutex
	key  []byte
	path string // "" when the key came from a flag/env (not a file)
}

// NewManager builds a Manager from a key. path is the key file to persist to on
// Adopt, or "" if the key is supplied out-of-band (flag/env).
func NewManager(key []byte, path string) (*Manager, error) {
	c, err := New(key)
	if err != nil {
		return nil, err
	}
	return &Manager{cipher: c, key: clone(key), path: path}, nil
}

// Cipher returns the shared cipher (stable pointer; its key may change).
func (m *Manager) Cipher() *Cipher { return m.cipher }

// KeyHex returns the current key, hex-encoded (for inclusion in a backup).
func (m *Manager) KeyHex() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return hex.EncodeToString(m.key)
}

// PersistsToFile reports whether the key is stored in a file (vs flag/env).
func (m *Manager) PersistsToFile() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.path != ""
}

// Adopt switches to a new key: re-keys the live cipher and, when the key is
// file-backed, writes it to disk so it survives a restart.
//
// Ordering matters for consistency: the key is validated and durably persisted
// *before* the in-memory cipher/key are changed, so a failed disk write leaves
// the live key and the on-disk key in agreement (both unchanged).
func (m *Manager) Adopt(key []byte) error {
	if _, err := newAEAD(key); err != nil { // validate before mutating anything
		return err
	}
	m.mu.Lock()
	path := m.path
	m.mu.Unlock()
	if path != "" {
		if err := os.WriteFile(path, []byte(hex.EncodeToString(key)), 0o600); err != nil {
			return fmt.Errorf("persisting adopted key: %w", err)
		}
	}
	// Key already validated above, so SetKey cannot fail here.
	_ = m.cipher.SetKey(key)
	m.mu.Lock()
	m.key = clone(key)
	m.mu.Unlock()
	return nil
}

func clone(b []byte) []byte {
	out := make([]byte, len(b))
	copy(out, b)
	return out
}
