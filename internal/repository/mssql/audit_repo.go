package mssql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/your-org/datavault/internal/domain/model"
)

type msAuditRepo struct{ db *sql.DB }

func (a *msAuditRepo) Append(ctx context.Context, event *model.AuditEvent) error {
	_, err := a.db.ExecContext(ctx, `
		INSERT INTO audit_log (id, tenant_id, operation, record_id, actor, ip_address, status, detail, created_at)
		VALUES (@p1,@p2,@p3,@p4,@p5,@p6,@p7,@p8,@p9)`,
		event.ID, event.TenantID, event.Operation, event.RecordID,
		event.Actor, event.IPAddress, event.Status, event.Detail, event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("audit append: %w", err)
	}
	return nil
}
