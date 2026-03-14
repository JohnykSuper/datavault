package crypto

// Zeroize overwrites a byte slice with zeros to reduce the window during which
// a plaintext DEK resides in memory. Call deferred immediately after obtaining
// any plaintext key material.
//
//	dek, _ := hsm.UnwrapDEK(...)
//	defer crypto.Zeroize(dek)
func Zeroize(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
