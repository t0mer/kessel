// Package crypto provides AES-256-GCM encryption for secrets at rest.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sync"
)

// KeySize is the AES-256 key length in bytes.
const KeySize = 32

// NewKey generates a random 32-byte key.
func NewKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generating key: %w", err)
	}
	return key, nil
}

// ParseKey decodes a hex-encoded 32-byte key.
func ParseKey(hexKey string) ([]byte, error) {
	b, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("decoding hex key: %w", err)
	}
	if len(b) != KeySize {
		return nil, fmt.Errorf("key must be %d bytes, got %d", KeySize, len(b))
	}
	return b, nil
}

func newAEAD(key []byte) (cipher.AEAD, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("key must be %d bytes, got %d", KeySize, len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating AES cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}
	return aead, nil
}

// Cipher encrypts and decrypts with AES-256-GCM. Its key can be swapped at
// runtime (during a restore) via SetKey; the handle is safe for concurrent use.
type Cipher struct {
	mu   sync.RWMutex
	aead cipher.AEAD
}

// New builds a Cipher from a 32-byte key.
func New(key []byte) (*Cipher, error) {
	aead, err := newAEAD(key)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}

// SetKey replaces the cipher's key atomically.
func (c *Cipher) SetKey(key []byte) error {
	aead, err := newAEAD(key)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.aead = aead
	c.mu.Unlock()
	return nil
}

func (c *Cipher) ref() cipher.AEAD {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.aead
}

// Encrypt returns nonce||ciphertext.
func (c *Cipher) Encrypt(plaintext []byte) ([]byte, error) {
	aead := c.ref()
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}
	return aead.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt reverses Encrypt.
func (c *Cipher) Decrypt(ciphertext []byte) ([]byte, error) {
	aead := c.ref()
	ns := aead.NonceSize()
	if len(ciphertext) < ns {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ct := ciphertext[:ns], ciphertext[ns:]
	pt, err := aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypting: %w", err)
	}
	return pt, nil
}

// LoadOrCreateKey reads a hex key from path, or generates and persists one
// (0600) if the file does not exist.
func LoadOrCreateKey(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return ParseKey(string(trimSpace(data)))
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading key file: %w", err)
	}
	key, err := NewKey()
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(hex.EncodeToString(key)), 0o600); err != nil {
		return nil, fmt.Errorf("writing key file: %w", err)
	}
	return key, nil
}

func trimSpace(b []byte) []byte {
	start, end := 0, len(b)
	for start < end && isSpace(b[start]) {
		start++
	}
	for end > start && isSpace(b[end-1]) {
		end--
	}
	return b[start:end]
}

func isSpace(c byte) bool { return c == ' ' || c == '\n' || c == '\r' || c == '\t' }
