// Package oracle implements the DataVault repository ports using go-ora (pure Go).
// All SQL uses Oracle syntax (:1 named/positional params, ROWNUM/OFFSET FETCH).
// Do not mix with PostgreSQL ($1) or MSSQL (@p1) placeholders.
package oracle

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/sijms/go-ora/v2"
	"github.com/your-org/datavault/internal/config"
	"github.com/your-org/datavault/internal/domain/model"
	"github.com/your-org/datavault/internal/domain/port"
)

// New opens a sql.DB pool for Oracle and returns record + audit repositories.
// DSN format: oracle://user:pass@host:port/service
func New(cfg *config.Config) (port.RecordRepository, port.AuditRepository, error) {
	dsn := fmt.Sprintf("oracle://%s:%s@%s", cfg.OracleUser, cfg.OraclePass, cfg.OracleDSN)
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, nil, err
	}
	if err := db.PingContext(context.Background()); err != nil {
		return nil, nil, fmt.Errorf("oracle ping: %w", err)
	}
	return &oraRecordRepo{db: db}, &oraAuditRepo{db: db}, nil
}

// ── Record repository ────────────────────────────────────────────────────────

type oraRecordRepo struct{ db *sql.DB }

// Ping implements port.Pinger — used by the readiness probe.
func (r *oraRecordRepo) Ping(ctx context.Context) error { return r.db.PingContext(ctx) }

func (r *oraRecordRepo) Save(ctx context.Context, rec *model.Record) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.ExecContext(ctx, `
		INSERT INTO records (id, tenant_id, ciphertext, nonce, aad, wrapped_dek, key_version, created_at, updated_at)
		VALUES (:1,:2,:3,:4,:5,:6,:7,SYSTIMESTAMP,SYSTIMESTAMP)`,
		rec.ID, rec.TenantID, rec.Ciphertext, rec.Nonce, rec.AAD, rec.WrappedDEK, rec.KeyVersion,
	)
	if err != nil {
		return fmt.Errorf("insert record: %w", err)
	}

	for _, token := range rec.SearchTokens {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO search_tokens (record_id, tenant_id, token) VALUES (:1,:2,:3)`,
			rec.ID, rec.TenantID, token,
		)
		if err != nil {
			return fmt.Errorf("insert search token: %w", err)
		}
	}

	return tx.Commit()
}

func (r *oraRecordRepo) FindByID(ctx context.Context, tenantID, id string) (*model.Record, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, ciphertext, nonce, aad, wrapped_dek, key_version, created_at, updated_at
		FROM records WHERE id=:1 AND tenant_id=:2`, id, tenantID)

	var rec model.Record
	if err := row.Scan(&rec.ID, &rec.TenantID, &rec.Ciphertext, &rec.Nonce, &rec.AAD,
		&rec.WrappedDEK, &rec.KeyVersion, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
		return nil, fmt.Errorf("FindByID: %w", err)
	}
	return &rec, nil
}

func (r *oraRecordRepo) FindBySearchToken(ctx context.Context, tenantID, token string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT record_id FROM search_tokens WHERE tenant_id=:1 AND token=:2`,
		tenantID, token)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *oraRecordRepo) ListByKeyVersion(ctx context.Context, tenantID string, version, limit, offset int) ([]*model.Record, error) {
	// Oracle 12c+ OFFSET/FETCH NEXT syntax.
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, ciphertext, nonce, aad, wrapped_dek, key_version, created_at, updated_at
		FROM records WHERE tenant_id=:1 AND key_version=:2
		ORDER BY created_at
		OFFSET :3 ROWS FETCH NEXT :4 ROWS ONLY`,
		tenantID, version, offset, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*model.Record
	for rows.Next() {
		var rec model.Record
		if err := rows.Scan(&rec.ID, &rec.TenantID, &rec.Ciphertext, &rec.Nonce, &rec.AAD,
			&rec.WrappedDEK, &rec.KeyVersion, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
			return nil, err
		}
		records = append(records, &rec)
	}
	return records, rows.Err()
}

func (r *oraRecordRepo) Update(ctx context.Context, rec *model.Record) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE records SET wrapped_dek=:1, key_version=:2, updated_at=SYSTIMESTAMP
		WHERE id=:3 AND tenant_id=:4`,
		rec.WrappedDEK, rec.KeyVersion, rec.ID, rec.TenantID)
	return err
}
