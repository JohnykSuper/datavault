package model

import "time"

// KeyVersion represents a versioned KEK/DEK generation. Versions are
// monotonically increasing; the highest active version is used for new
// encryptions. Older versions are kept for decryption (rewrap migrates them).
type KeyVersion struct {
	Version   int       `db:"version"`
	TenantID  string    `db:"tenant_id"`
	Active    bool      `db:"active"`
	CreatedAt time.Time `db:"created_at"`
}

// WrappedDEK is a data-encryption key that has been wrapped (encrypted) by
// the HSM using the tenant's KEK. Only the wrapped form is ever persisted.
type WrappedDEK struct {
	KeyVersion int
	Data       []byte // opaque blob returned by HSM WrapKey
}
