// Package port defines the outbound interfaces (driven ports) that the domain
// service requires from infrastructure adapters.
package port

import (
	"context"

	"github.com/your-org/datavault/internal/domain/model"
)

// RecordRepository is the persistence port for encrypted records.
type RecordRepository interface {
	Save(ctx context.Context, r *model.Record) error
	FindByID(ctx context.Context, tenantID, id string) (*model.Record, error)
	// FindBySearchToken returns record IDs that match the given HMAC token.
	FindBySearchToken(ctx context.Context, tenantID, token string) ([]string, error)
	// ListByKeyVersion returns records encrypted under a specific key version
	// (used during rewrap migrations).
	ListByKeyVersion(ctx context.Context, tenantID string, version int, limit, offset int) ([]*model.Record, error)
	Update(ctx context.Context, r *model.Record) error
}

// AuditRepository persists immutable audit log entries.
type AuditRepository interface {
	Append(ctx context.Context, event *model.AuditEvent) error
}

// Pinger is implemented by any infrastructure component that can verify
// its own connectivity (e.g. a database pool). Used by the readiness probe.
type Pinger interface {
	Ping(ctx context.Context) error
}
