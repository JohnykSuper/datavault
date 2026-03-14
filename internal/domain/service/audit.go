package service

import (
	"context"

	"time"

	"github.com/google/uuid"
	"github.com/your-org/datavault/internal/domain/model"
)

// writeAudit appends an audit event asynchronously (best-effort).
// Errors are logged but do not fail the main operation.
func (s *Service) writeAudit(
	ctx context.Context,
	tenantID, recordID, operation, actor, ip, status, detail string,
) {
	event := &model.AuditEvent{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Operation: operation,
		RecordID:  recordID,
		Actor:     actor,
		IPAddress: ip,
		Status:    status,
		Detail:    detail,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.audit.Append(ctx, event); err != nil {
		// Audit failures are logged but must not disrupt the main request.
		s.log.Error("audit write failed", "operation", operation, "error", err)
	}
}
