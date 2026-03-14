package port

import "context"

// HSM is the outbound port for the Hardware Security Module.
// It is used exclusively for KEK-level operations (wrap/unwrap DEK).
// Bulk data encryption must NOT use HSM — use crypto.EncryptAESGCM instead.
type HSM interface {
	// WrapDEK encrypts a plaintext DEK under the current KEK for the given
	// tenant and key version. Returns an opaque wrapped blob.
	WrapDEK(ctx context.Context, tenantID string, keyVersion int, plaintextDEK []byte) ([]byte, error)

	// UnwrapDEK decrypts a previously wrapped DEK blob. The caller must
	// zeroize the returned slice after use.
	UnwrapDEK(ctx context.Context, tenantID string, keyVersion int, wrappedDEK []byte) ([]byte, error)

	// CurrentKeyVersion returns the active key version for a tenant.
	CurrentKeyVersion(ctx context.Context, tenantID string) (int, error)
}
