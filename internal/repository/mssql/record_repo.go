// Package mssql implements the DataVault repository ports using go-mssqldb.
// All SQL uses MSSQL syntax (e.g. TOP, OFFSET/FETCH). Do not mix with
// PostgreSQL ($1) or Oracle (:name) placeholders.
package mssql

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/microsoft/go-mssqldb"
	"github.com/your-org/datavault/internal/config"
	"github.com/your-org/datavault/internal/domain/model"
	"github.com/your-org/datavault/internal/domain/port"
)

// New opens a sql.DB pool for MSSQL and returns record + audit repositories.
//
// Pool parameters are read from config (DATAVAULT_DB_* env vars).
// database/sql automatically retries with a fresh connection on ErrBadConn.
func New(cfg *config.Config) (port.RecordRepository, port.AuditRepository, error) {
	dsn := fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s",
		cfg.MSSQLUser, cfg.MSSQLPass, cfg.MSSQLHost, cfg.MSSQLPort, cfg.MSSQLDB)
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return nil, nil, err
	}
	db.SetMaxOpenConns(cfg.DBMaxConns)
	db.SetMaxIdleConns(cfg.DBMinConns)
	db.SetConnMaxLifetime(cfg.DBConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.DBConnMaxIdleTime)
	if err := db.PingContext(context.Background()); err != nil {
		return nil, nil, fmt.Errorf("mssql ping: %w", err)
	}
	return &msRecordRepo{db: db}, &msAuditRepo{db: db}, nil
}

// ── Record repository ────────────────────────────────────────────────────────

type msRecordRepo struct{ db *sql.DB }

// Ping implements port.Pinger — used by the readiness probe.
func (r *msRecordRepo) Ping(ctx context.Context) error { return r.db.PingContext(ctx) }

func (r *msRecordRepo) Save(ctx context.Context, rec *model.Record) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.ExecContext(ctx, `
		INSERT INTO records (id, tenant_id, ciphertext, nonce, aad, wrapped_dek, key_version, created_at, updated_at)
		VALUES (@p1,@p2,@p3,@p4,@p5,@p6,@p7,GETUTCDATE(),GETUTCDATE())`,
		rec.ID, rec.TenantID, rec.Ciphertext, rec.Nonce, rec.AAD, rec.WrappedDEK, rec.KeyVersion,
	)
	if err != nil {
		return fmt.Errorf("insert record: %w", err)
	}

	for _, token := range rec.SearchTokens {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO search_tokens (record_id, tenant_id, token) VALUES (@p1,@p2,@p3)`,
			rec.ID, rec.TenantID, token,
		)
		if err != nil {
			return fmt.Errorf("insert search token: %w", err)
		}
	}

	return tx.Commit()
}

func (r *msRecordRepo) FindByID(ctx context.Context, tenantID, id string) (*model.Record, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, ciphertext, nonce, aad, wrapped_dek, key_version, created_at, updated_at
		FROM records WHERE id=@p1 AND tenant_id=@p2`, id, tenantID)

	var rec model.Record
	if err := row.Scan(&rec.ID, &rec.TenantID, &rec.Ciphertext, &rec.Nonce, &rec.AAD,
		&rec.WrappedDEK, &rec.KeyVersion, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
		return nil, fmt.Errorf("FindByID: %w", err)
	}
	return &rec, nil
}

func (r *msRecordRepo) FindBySearchToken(ctx context.Context, tenantID, token string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT record_id FROM search_tokens WHERE tenant_id=@p1 AND token=@p2`,
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

func (r *msRecordRepo) ListByKeyVersion(ctx context.Context, tenantID string, version, limit, offset int) ([]*model.Record, error) {
	// MSSQL 2012+ OFFSET/FETCH syntax.
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, ciphertext, nonce, aad, wrapped_dek, key_version, created_at, updated_at
		FROM records WHERE tenant_id=@p1 AND key_version=@p2
		ORDER BY created_at
		OFFSET @p3 ROWS FETCH NEXT @p4 ROWS ONLY`,
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

func (r *msRecordRepo) Update(ctx context.Context, rec *model.Record) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE records SET wrapped_dek=@p1, key_version=@p2, updated_at=GETUTCDATE()
		WHERE id=@p3 AND tenant_id=@p4`,
		rec.WrappedDEK, rec.KeyVersion, rec.ID, rec.TenantID)
	return err
}
