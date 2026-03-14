-- DataVault PostgreSQL migration: initial schema
-- Run via: make migrate-postgres

BEGIN;

CREATE TABLE IF NOT EXISTS records (
    id          UUID PRIMARY KEY,
    tenant_id   VARCHAR(128) NOT NULL,
    ciphertext  BYTEA        NOT NULL,
    nonce       BYTEA        NOT NULL,  -- 12 bytes, AES-GCM nonce
    aad         BYTEA,
    wrapped_dek BYTEA        NOT NULL,  -- DEK wrapped by HSM KEK
    key_version INTEGER      NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_records_tenant_version ON records (tenant_id, key_version);

CREATE TABLE IF NOT EXISTS search_tokens (
    id        BIGSERIAL    PRIMARY KEY,
    record_id UUID         NOT NULL REFERENCES records(id) ON DELETE CASCADE,
    tenant_id VARCHAR(128) NOT NULL,
    token     CHAR(64)     NOT NULL  -- hex-encoded HMAC-SHA256
);

CREATE INDEX IF NOT EXISTS idx_search_tokens_lookup ON search_tokens (tenant_id, token);

CREATE TABLE IF NOT EXISTS audit_log (
    id         UUID         PRIMARY KEY,
    tenant_id  VARCHAR(128) NOT NULL,
    operation  VARCHAR(32)  NOT NULL,  -- encrypt | decrypt | search | rewrap
    record_id  UUID,
    actor      VARCHAR(255),
    ip_address VARCHAR(45),
    status     VARCHAR(16)  NOT NULL,  -- success | failure
    detail     TEXT,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_tenant_time ON audit_log (tenant_id, created_at DESC);

COMMIT;
