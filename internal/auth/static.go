// Package auth provides authentication utilities for DataVault.
package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
)

// entry associates a SHA-256-hashed token with its actor name.
type entry struct {
	tokenHash [sha256.Size]byte
	actor     string
}

// StaticValidator authenticates bearer tokens against a fixed set of
// key → actor mappings loaded from configuration. Comparison is done with
// constant-time equality to prevent timing attacks.
//
// The raw token values are never stored in memory after construction — only
// their SHA-256 digests are kept.
type StaticValidator struct {
	entries []entry
}

// NewStaticValidator builds a StaticValidator from a map of rawToken → actorName.
// rawKeys must not be empty.
func NewStaticValidator(rawKeys map[string]string) (*StaticValidator, error) {
	if len(rawKeys) == 0 {
		return nil, fmt.Errorf("auth: at least one API key must be configured")
	}
	v := &StaticValidator{}
	for token, actor := range rawKeys {
		if token == "" || actor == "" {
			return nil, fmt.Errorf("auth: token and actor name must not be empty")
		}
		v.entries = append(v.entries, entry{
			tokenHash: sha256.Sum256([]byte(token)),
			actor:     actor,
		})
	}
	return v, nil
}

// Validate checks the bearer token against all registered entries using
// constant-time comparison. Returns the actor name on success, or an error
// if the token does not match any entry.
//
// The incoming token is never logged.
func (v *StaticValidator) Validate(_ context.Context, token string) (string, error) {
	incoming := sha256.Sum256([]byte(token))
	for _, e := range v.entries {
		if subtle.ConstantTimeCompare(incoming[:], e.tokenHash[:]) == 1 {
			return e.actor, nil
		}
	}
	return "", fmt.Errorf("auth: invalid API key")
}
