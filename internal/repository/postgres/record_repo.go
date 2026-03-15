// Package postgres implements the DataVault repository ports using pgx v5.
// All SQL is explicit — no ORM. Do not add MSSQL or Oracle syntax here.
package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/datavault/internal/config"
	"github.com/your-org/datavault/internal/domain/model"
	"github.com/your-org/datavault/internal/domain/port"
)

// New opens a pgx connection pool and returns record + audit repositories.
//
// Pool settings:
//   - MinConns=2, MaxConns=20 (prevents thundering herd on cold start)
//   - MaxConnLifetime=30min (recycles long-lived connections, handles silent drops)
//   - MaxConnIdleTime=5min (releases idle capacity quickly)
//   - HealthCheckPeriod=1min (proactively pings idle connections; bad ones are dropped
//     and replaced automatically — this is the primary reconnect mechanism)
func New(cfg *config.Config) (port.RecordRepository, port.AuditRepository, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("pgxpool.ParseConfig: %w", err)
	}
	poolCfg.MinConns = 2
	poolCfg.MaxConns = 20
	poolCfg.MaxConnLifetime = 30 * time.Minute
	poolCfg.MaxConnIdleTime = 5 * time.Minute
	poolCfg.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("pgxpool.NewWithConfig: %w", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		return nil, nil, fmt.Errorf("postgres ping: %w", err)
	}
	return &pgRecordRepo{pool: pool}, &pgAuditRepo{pool: pool}, nil
}

// ── Record repository ────────────────────────────────────────────────────────

type pgRecordRepo struct{ pool *pgxpool.Pool }

// Ping implements port.Pinger — used by the readiness probe.
func (r *pgRecordRepo) Ping(ctx context.Context) error { return r.pool.Ping(ctx) }

func (r *pgRecordRepo) Save(ctx context.Context, rec *model.Record) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `
		INSERT INTO records (id, tenant_id, ciphertext, nonce, aad, wrapped_dek, key_version, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,NOW(),NOW())`,
		rec.ID, rec.TenantID, rec.Ciphertext, rec.Nonce, rec.AAD, rec.WrappedDEK, rec.KeyVersion,
	)
	if err != nil {
		return fmt.Errorf("insert record: %w", err)
	}

	for _, token := range rec.SearchTokens {
		_, err = tx.Exec(ctx,
			`INSERT INTO search_tokens (record_id, tenant_id, token) VALUES ($1,$2,$3)`,
			rec.ID, rec.TenantID, token,
		)
		if err != nil {
			return fmt.Errorf("insert search token: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *pgRecordRepo) FindByID(ctx context.Context, tenantID, id string) (*model.Record, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, ciphertext, nonce, aad, wrapped_dek, key_version, created_at, updated_at
		FROM records WHERE id=$1 AND tenant_id=$2`, id, tenantID)

	var rec model.Record
	if err := row.Scan(&rec.ID, &rec.TenantID, &rec.Ciphertext, &rec.Nonce, &rec.AAD,
		&rec.WrappedDEK, &rec.KeyVersion, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
		return nil, fmt.Errorf("FindByID: %w", err)
	}
	return &rec, nil
}

func (r *pgRecordRepo) FindBySearchToken(ctx context.Context, tenantID, token string) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT record_id FROM search_tokens WHERE tenant_id=$1 AND token=$2`,
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

func (r *pgRecordRepo) ListByKeyVersion(ctx context.Context, tenantID string, version, limit, offset int) ([]*model.Record, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, ciphertext, nonce, aad, wrapped_dek, key_version, created_at, updated_at
		FROM records WHERE tenant_id=$1 AND key_version=$2 ORDER BY created_at LIMIT $3 OFFSET $4`,
		tenantID, version, limit, offset)
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

func (r *pgRecordRepo) Update(ctx context.Context, rec *model.Record) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE records SET wrapped_dek=$1, key_version=$2, updated_at=NOW()
		WHERE id=$3 AND tenant_id=$4`,
		rec.WrappedDEK, rec.KeyVersion, rec.ID, rec.TenantID)
	return err
}

// ── Unused import guard ──────────────────────────────────────────────────────
var _ = strings.Join
