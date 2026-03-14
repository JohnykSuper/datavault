package hsm

import (
	"bytes"
	"context"
	"testing"
)

// TestRFC3394WrapUnwrapRoundTrip verifies that aesWrap / aesUnwrap are inverses.
func TestRFC3394WrapUnwrapRoundTrip(t *testing.T) {
	kek := make([]byte, 32) // AES-256 KEK (zero bytes, fine for testing)
	dek := make([]byte, 32) // 256-bit DEK
	for i := range dek {
		dek[i] = byte(i + 1)
	}

	wrapped, err := aesWrap(kek, dek)
	if err != nil {
		t.Fatalf("aesWrap: %v", err)
	}
	// wrapped output is len(dek)+8 = 40 bytes
	if len(wrapped) != len(dek)+8 {
		t.Fatalf("expected %d bytes, got %d", len(dek)+8, len(wrapped))
	}

	unwrapped, err := aesUnwrap(kek, wrapped)
	if err != nil {
		t.Fatalf("aesUnwrap: %v", err)
	}
	if !bytes.Equal(unwrapped, dek) {
		t.Fatalf("unwrapped DEK does not match original\nwant: %x\ngot:  %x", dek, unwrapped)
	}
}

// TestRFC3394WrongKEK ensures integrity check fails when the KEK is wrong.
func TestRFC3394WrongKEK(t *testing.T) {
	kek := make([]byte, 32)
	for i := range kek {
		kek[i] = 0xAA
	}
	dek := make([]byte, 32)
	for i := range dek {
		dek[i] = byte(i)
	}

	wrapped, err := aesWrap(kek, dek)
	if err != nil {
		t.Fatalf("aesWrap: %v", err)
	}

	wrongKEK := make([]byte, 32)
	for i := range wrongKEK {
		wrongKEK[i] = 0xBB
	}
	if _, err := aesUnwrap(wrongKEK, wrapped); err == nil {
		t.Fatal("expected integrity error with wrong KEK, got nil")
	}
}

// TestRFC3394InvalidInputs verifies error paths for bad input sizes.
func TestRFC3394InvalidInputs(t *testing.T) {
	kek := make([]byte, 32)

	// DEK too short
	if _, err := aesWrap(kek, make([]byte, 8)); err == nil {
		t.Fatal("expected error for 8-byte DEK, got nil")
	}
	// DEK not multiple of 8
	if _, err := aesWrap(kek, make([]byte, 17)); err == nil {
		t.Fatal("expected error for 17-byte DEK, got nil")
	}
	// Wrapped too short
	if _, err := aesUnwrap(kek, make([]byte, 16)); err == nil {
		t.Fatal("expected error for too-short wrapped input, got nil")
	}
}

// TestStubEncryptDecryptRoundTrip tests the full Stub HSM via public interface.
func TestStubEncryptDecryptRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := NewStub()
	tenant := "tenant-1"

	ver, err := s.CurrentKeyVersion(ctx, tenant)
	if err != nil {
		t.Fatalf("CurrentKeyVersion: %v", err)
	}
	if ver != 1 {
		t.Fatalf("expected initial version 1, got %d", ver)
	}

	dek := make([]byte, 32)
	for i := range dek {
		dek[i] = byte(i)
	}

	wrapped, err := s.WrapDEK(ctx, tenant, ver, dek)
	if err != nil {
		t.Fatalf("WrapDEK: %v", err)
	}

	got, err := s.UnwrapDEK(ctx, tenant, ver, wrapped)
	if err != nil {
		t.Fatalf("UnwrapDEK: %v", err)
	}
	if !bytes.Equal(got, dek) {
		t.Fatalf("unwrapped DEK mismatch\nwant: %x\ngot:  %x", dek, got)
	}
}
