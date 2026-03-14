package model

import "time"

// AuditEvent captures an immutable audit log entry for every operation.
// Sensitive values (plaintext, DEK, HMAC key) must NEVER appear here.
type AuditEvent struct {
	ID        string    `db:"id"`
	TenantID  string    `db:"tenant_id"`
	Operation string    `db:"operation"` // encrypt | decrypt | search | rewrap
	RecordID  string    `db:"record_id"`
	Actor     string    `db:"actor"` // caller identity (API key id, service account, etc.)
	IPAddress string    `db:"ip_address"`
	Status    string    `db:"status"` // success | failure
	Detail    string    `db:"detail"` // non-sensitive context only
	CreatedAt time.Time `db:"created_at"`
}
