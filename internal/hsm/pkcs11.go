package hsm

import (
	"context"
	"fmt"
)

// PKCS11 is the production HSM adapter using the PKCS#11 standard interface.
//
// TODO: Implement using github.com/miekg/pkcs11 or ThalesIgnite/crypto11.
// Required configuration (from config.Config):
//   - PKCS11_LIB  — path to the vendor's PKCS#11 shared library
//   - PKCS11_SLOT — slot index or slot ID
//   - PKCS11_PIN  — user PIN (loaded from secret manager, never logged)
//
// Key mapping convention:
//   - KEK label: "datavault-kek-{tenantID}-v{keyVersion}"
//   - Keys are pre-generated in the HSM; this adapter only wraps/unwraps.
type PKCS11 struct {
	// TODO: add pkcs11.Ctx and session pool fields
}

func (p *PKCS11) CurrentKeyVersion(_ context.Context, _ string) (int, error) {
	return 0, fmt.Errorf("PKCS11: not implemented — see internal/hsm/pkcs11.go")
}

func (p *PKCS11) WrapDEK(_ context.Context, _, _ string, _, _ []byte) ([]byte, error) {
	// TODO: C_WrapKey with CKM_AES_KEY_WRAP
	return nil, fmt.Errorf("PKCS11: not implemented")
}

// Note: method signature must match port.HSM interface exactly.
func (p *PKCS11) UnwrapDEK(_ context.Context, _ string, _ int, _ []byte) ([]byte, error) {
	// TODO: C_UnwrapKey with CKM_AES_KEY_WRAP
	return nil, fmt.Errorf("PKCS11: not implemented")
}
