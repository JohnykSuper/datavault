// Package crypto provides the low-level cryptographic primitives for DataVault.
// Only AES-256-GCM and HMAC-SHA256 are used — no custom algorithms.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
)

// EncryptAESGCM encrypts plaintext using AES-256-GCM with the provided nonce
// and additional authenticated data (AAD).
//
// Requirements:
//   - key must be exactly 32 bytes (AES-256).
//   - nonce must be exactly 12 bytes (standard GCM nonce size).
//   - aad may be nil or empty.
func EncryptAESGCM(key, nonce, plaintext, aad []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("EncryptAESGCM: key must be 32 bytes, got %d", len(key))
	}
	if len(nonce) != 12 {
		return nil, fmt.Errorf("EncryptAESGCM: nonce must be 12 bytes, got %d", len(nonce))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("EncryptAESGCM: create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("EncryptAESGCM: create GCM: %w", err)
	}

	// Seal appends ciphertext+tag to a nil dst.
	return gcm.Seal(nil, nonce, plaintext, aad), nil
}

// DecryptAESGCM decrypts and authenticates a ciphertext (with 16-byte GCM tag)
// produced by EncryptAESGCM. Returns an error if authentication fails.
func DecryptAESGCM(key, nonce, ciphertext, aad []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("DecryptAESGCM: key must be 32 bytes, got %d", len(key))
	}
	if len(nonce) != 12 {
		return nil, fmt.Errorf("DecryptAESGCM: nonce must be 12 bytes, got %d", len(nonce))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("DecryptAESGCM: create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("DecryptAESGCM: create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("DecryptAESGCM: authentication failed: %w", err)
	}

	return plaintext, nil
}
