// Package cache provides the in-memory DEK cache with TTL expiry.
package cache

import (
	"fmt"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/your-org/datavault/internal/crypto"
)

// DEKCache stores plaintext DEKs in memory, indexed by (tenantID, keyVersion).
// Entries expire after the configured TTL. On eviction the DEK is zeroized.
type DEKCache struct {
	c   *gocache.Cache
	ttl time.Duration
}

// NewDEKCache creates a new cache with the given TTL and a cleanup interval
// of 2× TTL.
func NewDEKCache(ttl time.Duration) *DEKCache {
	c := gocache.New(ttl, ttl*2)

	// Register eviction callback to zeroize DEK bytes on expiry.
	c.OnEvicted(func(_ string, value interface{}) {
		if dek, ok := value.([]byte); ok {
			crypto.Zeroize(dek)
		}
	})

	return &DEKCache{c: c, ttl: ttl}
}

func cacheKey(tenantID string, keyVersion int) string {
	return fmt.Sprintf("%s:%d", tenantID, keyVersion)
}

// Get returns the cached plaintext DEK, or false if not present / expired.
func (d *DEKCache) Get(tenantID string, keyVersion int) ([]byte, bool) {
	v, ok := d.c.Get(cacheKey(tenantID, keyVersion))
	if !ok {
		return nil, false
	}
	return v.([]byte), true
}

// Set stores a plaintext DEK. The cache takes ownership of the slice.
func (d *DEKCache) Set(tenantID string, keyVersion int, dek []byte) {
	d.c.Set(cacheKey(tenantID, keyVersion), dek, d.ttl)
}

// Delete removes a DEK entry and zeroizes it immediately.
func (d *DEKCache) Delete(tenantID string, keyVersion int) {
	key := cacheKey(tenantID, keyVersion)
	if v, ok := d.c.Get(key); ok {
		if dek, ok := v.([]byte); ok {
			crypto.Zeroize(dek)
		}
	}
	d.c.Delete(key)
}

// ItemCount returns the number of DEKs currently held in the cache
// (including items that have not yet been cleaned up).
func (d *DEKCache) ItemCount() int {
	return d.c.ItemCount()
}

// CacheTTL returns the configured Time-To-Live for cache entries.
func (d *DEKCache) CacheTTL() time.Duration {
	return d.ttl
}
