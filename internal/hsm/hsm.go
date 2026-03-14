// Package hsm provides HSM adapter implementations.
// The stub is used in development/testing only. Production deployments must
// use the pkcs11 adapter (see pkcs11.go).
package hsm

import (
	"fmt"

	"github.com/your-org/datavault/internal/config"
	"github.com/your-org/datavault/internal/domain/port"
)

// New returns the appropriate HSM implementation based on DATAVAULT_HSM_MODE.
//   - "stub"   — in-process software HSM (dev/test only)
//   - "pkcs11" — production PKCS#11 adapter (not yet implemented)
//
// SECURITY: using HSMMode="stub" with DATAVAULT_ENV="prod" is a fatal error.
// Silent fallback from real HSM to stub mode is forbidden.
func New(cfg *config.Config) (port.HSM, error) {
	switch cfg.HSMMode {
	case "stub":
		if cfg.Env == "prod" {
			return nil, fmt.Errorf(
				"DATAVAULT_HSM_MODE=stub is not permitted in production (DATAVAULT_ENV=prod). " +
					"Configure a real PKCS#11 HSM or use a non-production environment.",
			)
		}
		return NewStub(), nil

	case "pkcs11":
		// TODO: return NewPKCS11(cfg)
		return nil, fmt.Errorf("PKCS#11 HSM adapter is not yet implemented — see internal/hsm/pkcs11.go")

	default:
		return nil, fmt.Errorf(
			"unknown DATAVAULT_HSM_MODE %q: expected \"stub\" or \"pkcs11\"",
			cfg.HSMMode,
		)
	}
}
