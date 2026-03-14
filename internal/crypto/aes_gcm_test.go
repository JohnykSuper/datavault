package crypto

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func randomBytes(t *testing.T, n int) []byte {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	return b
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := randomBytes(t, 32)
	nonce := randomBytes(t, 12)
	plaintext := []byte("hello DataVault")
	aad := []byte("tenant-id:record-id")

	ct, err := EncryptAESGCM(key, nonce, plaintext, aad)
	if err != nil {
		t.Fatalf("EncryptAESGCM: %v", err)
	}
	if bytes.Equal(ct, plaintext) {
		t.Fatal("ciphertext must differ from plaintext")
	}

	got, err := DecryptAESGCM(key, nonce, ct, aad)
	if err != nil {
		t.Fatalf("DecryptAESGCM: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("want %q, got %q", plaintext, got)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key := randomBytes(t, 32)
	nonce := randomBytes(t, 12)
	ct, _ := EncryptAESGCM(key, nonce, []byte("secret"), nil)

	wrongKey := randomBytes(t, 32)
	if _, err := DecryptAESGCM(wrongKey, nonce, ct, nil); err == nil {
		t.Fatal("expected error with wrong key, got nil")
	}
}

func TestDecryptWrongAAD(t *testing.T) {
	key := randomBytes(t, 32)
	nonce := randomBytes(t, 12)
	ct, _ := EncryptAESGCM(key, nonce, []byte("secret"), []byte("aad1"))

	if _, err := DecryptAESGCM(key, nonce, ct, []byte("aad2")); err == nil {
		t.Fatal("expected authentication error with tampered AAD, got nil")
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	key := randomBytes(t, 32)
	nonce := randomBytes(t, 12)
	ct, _ := EncryptAESGCM(key, nonce, []byte("secret"), nil)

	ct[0] ^= 0xFF // flip bits
	if _, err := DecryptAESGCM(key, nonce, ct, nil); err == nil {
		t.Fatal("expected error with tampered ciphertext, got nil")
	}
}

func TestEncryptKeyLength(t *testing.T) {
	nonce := randomBytes(t, 12)
	if _, err := EncryptAESGCM(make([]byte, 16), nonce, []byte("x"), nil); err == nil {
		t.Fatal("expected error for 16-byte key (not AES-256)")
	}
	if _, err := EncryptAESGCM(make([]byte, 32), make([]byte, 16), []byte("x"), nil); err == nil {
		t.Fatal("expected error for 16-byte nonce (not 12)")
	}
}
