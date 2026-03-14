// Package model contains the core domain models for DataVault.
package model

import "time"

// Record represents an encrypted data record stored in the database.
type Record struct {
	ID           string    `db:"id"`
	TenantID     string    `db:"tenant_id"`
	Ciphertext   []byte    `db:"ciphertext"`  // AES-256-GCM encrypted payload
	Nonce        []byte    `db:"nonce"`       // 12-byte GCM nonce
	AAD          []byte    `db:"aad"`         // additional authenticated data
	WrappedDEK   []byte    `db:"wrapped_dek"` // DEK wrapped by HSM KEK
	KeyVersion   int       `db:"key_version"`
	SearchTokens []string  `db:"-"` // indexed externally in search_tokens table
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}
