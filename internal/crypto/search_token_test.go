package crypto

import (
	"crypto/rand"
	"testing"
)

func TestHMACSha256TokenDeterministic(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key) //nolint:errcheck

	value := []byte("john.doe@example.com")
	t1 := HMACSha256Token(key, value)
	t2 := HMACSha256Token(key, value)
	if t1 != t2 {
		t.Fatalf("HMAC token not deterministic: %q != %q", t1, t2)
	}
}

func TestHMACSha256TokenDifferentKeys(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	rand.Read(key1) //nolint:errcheck
	rand.Read(key2) //nolint:errcheck

	value := []byte("some-value")
	if HMACSha256Token(key1, value) == HMACSha256Token(key2, value) {
		t.Fatal("different keys produced the same HMAC token (collision)")
	}
}

func TestHMACSha256TokenDifferentValues(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key) //nolint:errcheck

	if HMACSha256Token(key, []byte("a")) == HMACSha256Token(key, []byte("b")) {
		t.Fatal("different values produced the same HMAC token (collision)")
	}
}

func TestHMACSha256TokenIsHex(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key) //nolint:errcheck
	token := HMACSha256Token(key, []byte("value"))
	// SHA-256 output is 32 bytes → 64 hex chars.
	if len(token) != 64 {
		t.Fatalf("expected 64-char hex token, got len=%d", len(token))
	}
	for _, c := range token {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Fatalf("non-hex character %q in token", c)
		}
	}
}
