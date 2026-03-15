// Package hsm provides HSM adapter implementations.
// The stub is used in development/testing only. Production deployments must
// use the certex or pkcs11 adapter.
package hsm

import (
	"fmt"

	"github.com/your-org/datavault/internal/config"
	"github.com/your-org/datavault/internal/domain/port"
)

// FullClient combines HSM crypto operations (port.HSM) and HSM monitoring
// operations (port.HSMMonitor). All adapters returned by New() implement both.
type FullClient interface {
	port.HSM
	port.HSMMonitor
}

// New returns the appropriate HSM implementation based on DATAVAULT_HSM_MODE.
//   - "stub"   — in-process software HSM (dev/test only)
//   - "certex" — CERTEX HSM ES REST adapter (monitoring confirmed; crypto pending vendor docs)
//   - "pkcs11" — production PKCS#11 adapter (not yet implemented)
//
// SECURITY: using HSMMode="stub" with DATAVAULT_ENV="prod" is a fatal error.
// Silent fallback from real HSM to stub mode is forbidden.
func New(cfg *config.Config) (FullClient, error) {
	switch cfg.HSMMode {
	case "stub":
		if cfg.Env == "prod" {
			return nil, fmt.Errorf(
				"DATAVAULT_HSM_MODE=stub is not permitted in production (DATAVAULT_ENV=prod). " +
					"Configure a real HSM (certex or pkcs11) or use a non-production environment.",
			)
		}
		return NewStub(), nil

	case "certex":
		if cfg.HSMBaseURL == "" {
			return nil, fmt.Errorf("DATAVAULT_HSM_URL is required for certex HSM mode")
		}
		return NewCertexREST(cfg.HSMBaseURL, cfg.HSMUser, cfg.HSMPass), nil

	case "pkcs11":
		// TODO: return NewPKCS11(cfg)
		return nil, fmt.Errorf("PKCS#11 HSM adapter is not yet implemented — see internal/hsm/pkcs11.go")

	default:
		return nil, fmt.Errorf(
			"unknown DATAVAULT_HSM_MODE %q: expected \"stub\", \"certex\" or \"pkcs11\"",
			cfg.HSMMode,
		)
	}
}
