-- DataVault MSSQL migration: initial schema
-- Run via: make migrate-mssql

IF NOT EXISTS (SELECT 1 FROM sysobjects WHERE name='records' AND xtype='U')
BEGIN
    CREATE TABLE records (
        id          UNIQUEIDENTIFIER PRIMARY KEY,
        tenant_id   NVARCHAR(128)    NOT NULL,
        ciphertext  VARBINARY(MAX)   NOT NULL,
        nonce       VARBINARY(16)    NOT NULL,   -- 12 bytes, AES-GCM nonce
        aad         VARBINARY(MAX),
        wrapped_dek VARBINARY(MAX)   NOT NULL,   -- DEK wrapped by HSM KEK
        key_version INT              NOT NULL,
        created_at  DATETIME2        NOT NULL DEFAULT GETUTCDATE(),
        updated_at  DATETIME2        NOT NULL DEFAULT GETUTCDATE()
    );

    CREATE INDEX idx_records_tenant_version ON records (tenant_id, key_version);
END;
GO

IF NOT EXISTS (SELECT 1 FROM sysobjects WHERE name='search_tokens' AND xtype='U')
BEGIN
    CREATE TABLE search_tokens (
        id        BIGINT IDENTITY(1,1) PRIMARY KEY,
        record_id UNIQUEIDENTIFIER     NOT NULL REFERENCES records(id) ON DELETE CASCADE,
        tenant_id NVARCHAR(128)        NOT NULL,
        token     CHAR(64)             NOT NULL  -- hex-encoded HMAC-SHA256
    );

    CREATE INDEX idx_search_tokens_lookup ON search_tokens (tenant_id, token);
END;
GO

IF NOT EXISTS (SELECT 1 FROM sysobjects WHERE name='audit_log' AND xtype='U')
BEGIN
    CREATE TABLE audit_log (
        id         UNIQUEIDENTIFIER PRIMARY KEY,
        tenant_id  NVARCHAR(128)    NOT NULL,
        operation  NVARCHAR(32)     NOT NULL,
        record_id  UNIQUEIDENTIFIER,
        actor      NVARCHAR(255),
        ip_address NVARCHAR(45),
        status     NVARCHAR(16)     NOT NULL,
        detail     NVARCHAR(MAX),
        created_at DATETIME2        NOT NULL DEFAULT GETUTCDATE()
    );

    CREATE INDEX idx_audit_tenant_time ON audit_log (tenant_id, created_at DESC);
END;
GO
