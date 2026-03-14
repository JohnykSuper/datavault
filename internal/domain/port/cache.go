package port

// DEKCache is the outbound port for the in-memory DEK cache.
// Implementations must enforce TTL expiry and must NOT persist entries to disk.
type DEKCache interface {
	// Get returns the plaintext DEK for (tenantID, keyVersion), or false if
	// not cached or expired.
	Get(tenantID string, keyVersion int) (plaintextDEK []byte, ok bool)

	// Set stores a plaintext DEK in the cache for the configured TTL.
	// The caller must NOT zeroize the slice after passing it here; the cache
	// takes ownership and is responsible for zeroizing on eviction.
	Set(tenantID string, keyVersion int, plaintextDEK []byte)

	// Delete explicitly removes a cached DEK (e.g. after rewrap).
	Delete(tenantID string, keyVersion int)
}
