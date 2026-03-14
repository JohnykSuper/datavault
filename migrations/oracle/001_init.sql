-- DataVault Oracle migration: initial schema
-- Run via: make migrate-oracle

CREATE TABLE records (
    id          VARCHAR2(36)     PRIMARY KEY,
    tenant_id   VARCHAR2(128)    NOT NULL,
    ciphertext  BLOB             NOT NULL,
    nonce       RAW(16)          NOT NULL,   -- 12 bytes, AES-GCM nonce
    aad         BLOB,
    wrapped_dek BLOB             NOT NULL,   -- DEK wrapped by HSM KEK
    key_version NUMBER(10)       NOT NULL,
    created_at  TIMESTAMP        DEFAULT SYSTIMESTAMP NOT NULL,
    updated_at  TIMESTAMP        DEFAULT SYSTIMESTAMP NOT NULL
);

CREATE INDEX idx_records_tenant_version ON records (tenant_id, key_version);

CREATE TABLE search_tokens (
    id        NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    record_id VARCHAR2(36)  NOT NULL REFERENCES records(id) ON DELETE CASCADE,
    tenant_id VARCHAR2(128) NOT NULL,
    token     CHAR(64)      NOT NULL   -- hex-encoded HMAC-SHA256
);

CREATE INDEX idx_search_tokens_lookup ON search_tokens (tenant_id, token);

CREATE TABLE audit_log (
    id         VARCHAR2(36)  PRIMARY KEY,
    tenant_id  VARCHAR2(128) NOT NULL,
    operation  VARCHAR2(32)  NOT NULL,
    record_id  VARCHAR2(36),
    actor      VARCHAR2(255),
    ip_address VARCHAR2(45),
    status     VARCHAR2(16)  NOT NULL,
    detail     CLOB,
    created_at TIMESTAMP     DEFAULT SYSTIMESTAMP NOT NULL
);

CREATE INDEX idx_audit_tenant_time ON audit_log (tenant_id, created_at DESC);
