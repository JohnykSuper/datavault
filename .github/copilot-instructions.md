# DataVault – Copilot Instructions

## What Is DataVault
A production-grade Go service for **secure data encryption and key management** in a certification-authority environment. It uses **envelope encryption**: payloads are encrypted locally with AES-256-GCM (DEK), and the DEK is wrapped/unwrapped exclusively by an HSM (KEK). The service is stateless except for an in-memory DEK cache.

## Architecture

```
cmd/datavault/main.go          ← entry point, DI wiring
internal/
  config/        ← env-var config (Config struct, no hardcoded values)
  logger/        ← structured zap wrapper (never log DEK/plaintext/HMAC key)
  crypto/        ← AES-256-GCM (aes_gcm.go), HMAC-SHA256 search tokens (search_token.go), Zeroize (zeroize.go)
  domain/
    model/       ← Record, KeyVersion, AuditEvent
    port/        ← HSM, DEKCache, RecordRepository, AuditRepository interfaces
    service/     ← ALL business logic (encrypt, decrypt, search, rewrap, audit)
  hsm/           ← stub.go (dev, DATAVAULT_HSM_MODE=stub) + pkcs11.go (TODO: production, DATAVAULT_HSM_MODE=pkcs11)
  cache/         ← in-memory DEK cache with TTL + on-evict Zeroize
  repository/
    postgres/    ← pgx v5, $1 placeholders
    mssql/       ← go-mssqldb, @p1 placeholders, OFFSET/FETCH syntax
    oracle/      ← go-ora (pure Go), :1 placeholders, OFFSET ROWS FETCH syntax
  api/
    handler/     ← thin HTTP handlers — delegate immediately to service.*
    middleware/  ← StructuredLogger, APIKeyAuth
    router.go    ← chi router wiring
deploy/
  docker-compose.yml           ← postgres + migrate + datavault
  docker-compose.override.yml  ← dev extras: exposed ports, pgAdmin, debug log
docker/
  pgadmin-servers.json         ← pre-configured pgAdmin server entry
migrations/
  postgres/001_init.sql
  mssql/001_init.sql
  oracle/001_init.sql
Dockerfile                     ← multi-stage: builder (golang:1.23-alpine) + production (alpine)
.env / .env.example            ← all config via environment variables
```

## Critical Developer Workflows

```bash
make build            # go build → bin/datavault
make test             # go test ./... -v -count=1
make run              # go run ./cmd/datavault (requires .env loaded)
make tidy             # go mod tidy + go mod vendor
make up               # docker compose up -d --build  (from deploy/)
make down             # docker compose down
make logs             # follow datavault logs
make ps               # status of all containers
make migrate-postgres # apply migrations (requires DATAVAULT_DB_DSN)
make migrate-mssql    # apply migrations (requires DATAVAULT_MSSQL_* vars)
make migrate-oracle   # apply migrations (requires DATAVAULT_ORACLE_* vars)
make docker-build     # docker build --target production -t datavault:latest .
```

Copy `.env.example` → `.env` and set `DATAVAULT_HSM_MODE=stub` for local development.  
`HOST_PORT` (default `8081`) is the Docker host port; `DATAVAULT_APP_PORT` (default `8080`) is the in-container port.  
Docker Compose files live in `deploy/` — never in the project root.

### Environment Variable Naming
All app variables use the `DATAVAULT_` prefix per naming convention:  
`DATAVAULT_APP_PORT`, `DATAVAULT_DB_DRIVER`, `DATAVAULT_DB_DSN`, `DATAVAULT_HSM_MODE`,  
`DATAVAULT_SEARCH_KEY`, `DATAVAULT_DEK_CACHE_TTL`, `DATAVAULT_LOG_LEVEL`, `DATAVAULT_API_KEY`, `DATAVAULT_ENV`.

### Production Safety
`DATAVAULT_HSM_MODE=stub` is **fatal** when `DATAVAULT_ENV=prod`. There is no silent fallback to stub mode.

## Key Conventions

### API Endpoints
| Method | Path | Handler |
|--------|------|---------|
| POST | `/v1/encrypt` | `handler.Encrypt` |
| POST | `/v1/decrypt` | `handler.Decrypt` |
| GET | `/v1/search` | `handler.Search` |
| POST | `/v1/rewrap-dek` | `handler.Rewrap` |
| GET | `/health` | liveness probe |
| GET | `/ready` | readiness probe (DB ping) |

All `/v1/*` routes require `Authorization: Bearer <DATAVAULT_API_KEY>`.  
Decrypt accepts a JSON body `{"tenantId":"...","recordId":"..."}` (POST, not GET).

### JSON Field Names
All handler DTOs use **camelCase** JSON tags per naming convention:  
`tenantId`, `recordId`, `plaintextBase64`, `searchFields`, `keyVersion`, `recordIds`.

### Envelope Encryption Flow
1. Resolve current key version from HSM → check DEK cache → generate/unwrap DEK  
2. Encrypt payload: `crypto.EncryptAESGCM(dek, nonce, plaintext, aad)`  
3. Wrap DEK: `hsm.WrapDEK(...)` → store `wrapped_dek` in DB  
4. Always `defer crypto.Zeroize(dek)` immediately after obtaining plaintext DEK  

### SQL Dialect Rules — never mix placeholders
| Driver   | Placeholder | Date function   |
|----------|-------------|-----------------|
| postgres | `$1`        | `NOW()`         |
| mssql    | `@p1`       | `GETUTCDATE()`  |
| oracle   | `:1`        | `SYSTIMESTAMP`  |

### Handler Pattern
Handlers must stay thin: decode → call `svc.Method(ctx, Request{...})` → encode. Business logic goes in `internal/domain/service/`.

### Search Tokens
Never store the plaintext search value. Derive: `crypto.HMACSha256Token(cfg.HMACKey, []byte(value))` — env var `DATAVAULT_SEARCH_KEY`. Only the hex token hits the DB.

### HSM Integration Point
`internal/hsm/pkcs11.go` is the production TODO. Key label convention: `datavault-kek-{tenantID}-v{keyVersion}`. Use `C_WrapKey`/`C_UnwrapKey` with `CKM_AES_KEY_WRAP`.

### Logging Rules
- Use `log.Info/Error/Warn/Debug("message", "key", value, ...)` (key-value pairs)  
- **Never** log: `DEK`, `plaintext`, `HMACKey`, `wrappedDEK`, passwords, or API keys  
- Audit failures must be logged but must NOT fail the parent request  

## Adding a New Database Driver
1. Create `internal/repository/<driver>/record_repo.go` + `audit_repo.go`  
2. Implement `port.RecordRepository` and `port.AuditRepository`  
3. Add a case in `internal/repository/factory.go`  
4. Add a migration in `migrations/<driver>/001_init.sql`  
5. Add a `migrate-<driver>` target in `Makefile`  
