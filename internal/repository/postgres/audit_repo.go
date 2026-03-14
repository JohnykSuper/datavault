package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/datavault/internal/domain/model"
)

type pgAuditRepo struct{ pool *pgxpool.Pool }

func (a *pgAuditRepo) Append(ctx context.Context, event *model.AuditEvent) error {
	_, err := a.pool.Exec(ctx, `
		INSERT INTO audit_log (id, tenant_id, operation, record_id, actor, ip_address, status, detail, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		event.ID, event.TenantID, event.Operation, event.RecordID,
		event.Actor, event.IPAddress, event.Status, event.Detail, event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("audit append: %w", err)
	}
	return nil
}
