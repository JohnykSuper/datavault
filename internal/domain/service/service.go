// Package service contains the DataVault core business logic.
// Handlers must not embed business logic — delegate to this package.
package service

import (
	"github.com/your-org/datavault/internal/config"
	"github.com/your-org/datavault/internal/domain/port"
	"github.com/your-org/datavault/internal/logger"
)

// Service is the central application service that orchestrates all operations.
type Service struct {
	hsm     port.HSM
	cache   port.DEKCache
	records port.RecordRepository
	audit   port.AuditRepository
	log     *logger.Logger
	cfg     *config.Config
}

// New constructs a Service with all required dependencies injected.
func New(
	hsm port.HSM,
	cache port.DEKCache,
	records port.RecordRepository,
	audit port.AuditRepository,
	log *logger.Logger,
	cfg *config.Config,
) *Service {
	return &Service{
		hsm:     hsm,
		cache:   cache,
		records: records,
		audit:   audit,
		log:     log,
		cfg:     cfg,
	}
}
