// Package repository selects the appropriate database adapter based on config.
package repository

import (
	"fmt"

	"github.com/your-org/datavault/internal/config"
	"github.com/your-org/datavault/internal/domain/port"
	"github.com/your-org/datavault/internal/repository/mssql"
	"github.com/your-org/datavault/internal/repository/oracle"
	"github.com/your-org/datavault/internal/repository/postgres"
)

// Repositories bundles the two required repository ports plus a Pinger
// for the readiness probe.
type Repositories struct {
	Records port.RecordRepository
	Audit   port.AuditRepository
	Pinger  port.Pinger
}

// New instantiates the repository pair for the configured DATAVAULT_DB_DRIVER.
func New(cfg *config.Config) (*Repositories, error) {
	switch cfg.DBDriver {
	case "postgres":
		rec, aud, err := postgres.New(cfg)
		if err != nil {
			return nil, fmt.Errorf("postgres: %w", err)
		}
		return &Repositories{Records: rec, Audit: aud, Pinger: rec.(port.Pinger)}, nil

	case "mssql":
		rec, aud, err := mssql.New(cfg)
		if err != nil {
			return nil, fmt.Errorf("mssql: %w", err)
		}
		return &Repositories{Records: rec, Audit: aud, Pinger: rec.(port.Pinger)}, nil

	case "oracle":
		rec, aud, err := oracle.New(cfg)
		if err != nil {
			return nil, fmt.Errorf("oracle: %w", err)
		}
		return &Repositories{Records: rec, Audit: aud, Pinger: rec.(port.Pinger)}, nil

	default:
		return nil, fmt.Errorf("unknown DATAVAULT_DB_DRIVER: %q (expected postgres|mssql|oracle)", cfg.DBDriver)
	}
}
