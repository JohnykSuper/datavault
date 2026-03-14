package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// HMACSha256Token derives a deterministic search token from a value using
// HMAC-SHA256 with the service-wide HMAC key. The token is hex-encoded.
//
// Security note: the HMAC key must never be logged. The returned token is
// safe to store in the database as an opaque search index.
func HMACSha256Token(key, value []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(value)
	return hex.EncodeToString(mac.Sum(nil))
}
