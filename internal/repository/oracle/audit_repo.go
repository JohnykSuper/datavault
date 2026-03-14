package oracle

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/sijms/go-ora/v2"
	"github.com/your-org/datavault/internal/domain/model"
)

type oraAuditRepo struct{ db *sql.DB }

func (a *oraAuditRepo) Append(ctx context.Context, event *model.AuditEvent) error {
	_, err := a.db.ExecContext(ctx, `
		INSERT INTO audit_log (id, tenant_id, operation, record_id, actor, ip_address, status, detail, created_at)
		VALUES (:1,:2,:3,:4,:5,:6,:7,:8,:9)`,
		event.ID, event.TenantID, event.Operation, event.RecordID,
		event.Actor, event.IPAddress, event.Status, event.Detail, event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("audit append: %w", err)
	}
	return nil
}
