# DataVault – Copilot Instructions

## What Is DataVault
A production-grade Go service for **secure data encryption and key management** in a certification-authority environment. It uses **envelope encryption**: payloads are encrypted locally with AES-256-GCM (DEK), and the DEK is wrapped/unwrapped exclusively by an HSM (KEK). The service is stateless except for an in-memory DEK cache.

## Architecture

```
cmd/datavault/main.go          ← entry point, DI wiring
internal/
  config/        ← env-var config (Config struct, no hardcoded values)
  logger/        ← structured zap wrapper (never log DEK/plaintext/HMAC key)
  version/       ← version.Version string, set via ldflags at build time
  crypto/        ← AES-256-GCM (aes_gcm.go), HMAC-SHA256 search tokens (search_token.go), Zeroize (zeroize.go)
  domain/
    model/       ← Record, KeyVersion, AuditEvent
    port/        ← hsm.go (HSM crypto interface), hsm_monitor.go (HSMMonitor telemetry interface),
                   repository.go (RecordRepository, AuditRepository, Pinger),
                   dek_cache.go (DEKCache interface)
    service/     ← ALL business logic (encrypt, decrypt, search, rewrap, audit)
  hsm/
    hsm.go           ← factory: hsm.New() → FullClient (port.HSM + port.HSMMonitor)
    stub.go          ← dev-only in-process HSM (DATAVAULT_HSM_MODE=stub)
    certex_rest.go   ← CERTEX HSM ES REST adapter (DATAVAULT_HSM_MODE=certex)
    pkcs11.go        ← TODO: production PKCS#11 adapter (DATAVAULT_HSM_MODE=pkcs11)
  health/        ← Collector: unified /health snapshot (service + runtime + DB + HSM telemetry)
  cache/         ← in-memory DEK cache with TTL + on-evict Zeroize
  repository/
    postgres/    ← pgx v5, $1 placeholders
    mssql/       ← go-mssqldb, @p1 placeholders, OFFSET/FETCH syntax
    oracle/      ← go-ora (pure Go), :1 placeholders, OFFSET ROWS FETCH syntax
  api/
    handler/     ← thin HTTP handlers — delegate immediately to service.* or health.Collector
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

Database connection pool (all drivers, see `.env.example` for defaults):  
`DATAVAULT_DB_MAX_CONNS`, `DATAVAULT_DB_MIN_CONNS`, `DATAVAULT_DB_CONN_MAX_LIFETIME`,  
`DATAVAULT_DB_CONN_MAX_IDLE_TIME`, `DATAVAULT_DB_HEALTH_CHECK_PERIOD` (pgx only).

CERTEX HSM ES REST adapter (required when `DATAVAULT_HSM_MODE=certex`):  
`DATAVAULT_HSM_URL` (e.g. `https://10.0.0.1:8443`), `DATAVAULT_HSM_USER`, `DATAVAULT_HSM_PASS` — never log these values.

### Production Safety
`DATAVAULT_HSM_MODE=stub` is **fatal** when `DATAVAULT_ENV=prod`. There is no silent fallback to stub mode.  
Valid production modes: `certex` (CERTEX HSM ES REST) and `pkcs11` (PKCS#11, not yet implemented).

## Cryptographic Specification

| Parameter | Value |
|-----------|-------|
| Payload cipher | AES-256-GCM |
| Nonce | 12 bytes, random per record |
| Auth tag | 16 bytes |
| AAD | mandatory on every encrypt/decrypt call |
| DEK wrap | Development stub only; production mechanism depends on verified HSM integration |
| Search token | HMAC-SHA256, hex-encoded, derived from `DATAVAULT_SEARCH_KEY` |

No custom crypto. Use only Go standard library primitives (`crypto/aes`, `crypto/hmac`, `crypto/sha256`).
Production DEK wrap/unwrap must not be implemented from guesswork.
Use only:
- separately verified PKCS#11 documentation, or
- separately verified vendor SDK documentation, or
- separately verified CERTEX HSM ES crypto REST documentation

If verified crypto-wrap documentation is unavailable, keep production crypto-HSM integration marked as unsupported/TODO rather than fabricating it.

## Database Schema

### Required columns — main records table (`records` → target: `DV_SECURE_DATA`)

| Column | Postgres | MSSQL | Oracle |
|--------|----------|-------|--------|
| `id` | `UUID PRIMARY KEY` | `UNIQUEIDENTIFIER` | `VARCHAR2(36)` |
| `entity_type` | `VARCHAR` | `NVARCHAR` | `VARCHAR2` |
| `entity_id` | `VARCHAR` | `NVARCHAR` | `VARCHAR2` |
| `data_enc` | `BYTEA` | `VARBINARY(MAX)` | `BLOB` |
| `dek_wrapped` | `BYTEA` | `VARBINARY(MAX)` | `RAW(256)` |
| `nonce` | `BYTEA` | `VARBINARY(12)` | `RAW(12)` |
| `auth_tag` | `BYTEA` | `VARBINARY(16)` | `RAW(16)` |
| `alg` | `VARCHAR` | `NVARCHAR` | `VARCHAR2` |
| `kek_id` | `VARCHAR` | `NVARCHAR` | `VARCHAR2` |
| `key_version` | `INTEGER` | `INT` | `NUMBER` |
| `created_at` | `TIMESTAMPTZ` | `DATETIME2` | `TIMESTAMP WITH TIME ZONE` |
| `updated_at` | `TIMESTAMPTZ` | `DATETIME2` | `TIMESTAMP WITH TIME ZONE` |

### Search token table (`search_tokens` → target: `DV_SEARCH_TOKEN`)
`id`, `record_id` (FK), `field_name`, `token` (hex HMAC)

### Audit table (`audit_log` → target: `DV_AUDIT_LOG`)
`id`, `operation`, `entity_type`, `entity_id`, `result`, `error_code`, `duration_ms`, `created_at`

**Note:** current migrations use the legacy names (`records`, `search_tokens`, `audit_log`).  
Rename to `DV_*` names is a pending task (requires migration `002_rename_tables`).

## Database Type Reference

| Feature | Postgres | MSSQL | Oracle |
|---------|----------|-------|--------|
| Binary blob | `BYTEA` | `VARBINARY(MAX)` | `BLOB` / `RAW` |
| Timestamp | `TIMESTAMPTZ` | `DATETIME2` | `SYSTIMESTAMP` |
| Auto-return new ID | `RETURNING id` | `OUTPUT INSERTED.id` | sequence + `:id` OUT |
| Placeholder | `$1` | `@p1` | `:1` |

## Key Conventions

### API Endpoints
| Method | Path | Handler |
|--------|------|---------|
| POST | `/v1/encrypt` | `handler.Encrypt` |
| POST | `/v1/decrypt` | `handler.Decrypt` |
| GET | `/v1/search` | `handler.Search` |
| POST | `/v1/rewrap-dek` | `handler.Rewrap` |
| GET | `/health` | unified health probe: liveness + readiness + full telemetry |

**There is no separate `/ready` endpoint.** `GET /health` returns `200 ok` when all components (DB + HSM) are healthy and `503 error` when any component is unavailable. It always includes full telemetry: runtime metrics, memory stats, DB latency, HSM node counters, battery, NTP status, DEK cache state.

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

DataVault supports HSM integration through two complementary interfaces:

- **`port.HSM`** — cryptographic operations: `WrapDEK`, `UnwrapDEK`, `CurrentKeyVersion`, `Ping`
- **`port.HSMMonitor`** — operational telemetry: `NodeInfo`, `ClusterInfo`, `LogCount`, `Date`, `Battery`, `NTPStatus`, `ActiveKeys`

`hsm.FullClient` embeds both interfaces. `hsm.New(cfg)` always returns a `FullClient`. All adapters (`Stub`, `CertexREST`) implement both.

For CERTEX HSM ES, the currently verified integration surface from the available vendor document is an HTTPS REST API with Basic HTTP authorization, JSON request/response format, and POST requests.

Do not assume PKCS#11, `C_WrapKey`, `C_UnwrapKey`, `CKM_AES_KEY_WRAP`, or any cryptographic key-wrap endpoint unless a separate verified vendor document explicitly confirms them.

Implement HSM adapters as separate files:

- `internal/hsm/stub.go` — local development only (`DATAVAULT_HSM_MODE=stub`)
- `internal/hsm/certex_rest.go` — CERTEX HSM ES REST adapter (`DATAVAULT_HSM_MODE=certex`); monitoring fully implemented; crypto ops return `not implemented` pending vendor docs
- `internal/hsm/pkcs11.go` — TODO: production PKCS#11 adapter (`DATAVAULT_HSM_MODE=pkcs11`)

If a capability is not confirmed by vendor documentation, do not invent methods or fake production implementations.

### Logging Rules
- Use `log.Info/Error/Warn/Debug("message", "key", value, ...)` (key-value pairs)  
- **Never** log: `DEK`, `plaintext`, `HMACKey`, `wrappedDEK`, passwords, or API keys  
- Audit failures must be logged but must NOT fail the parent request  

## Coding Rules
- Do not mix SQL dialects or placeholders across drivers  
- Do not log sensitive data (DEK, plaintext, search key, API key, passwords)  
- Do not use mock/stub HSM in production (`DATAVAULT_ENV=prod` blocks it at startup)  
- Prefer explicit SQL over ORM  
- Keep port interfaces minimal and focused  
- Write unit tests for all crypto primitives  
- Use Go standard library where possible; minimise external dependencies  
- Use context timeouts on all DB and HSM calls  
- Service must remain stateless — no in-process mutable state except the DEK cache  
- Do not invent undocumented HSM endpoints or undocumented PKCS#11 capabilities
- For CERTEX HSM ES, only use vendor-documented REST endpoints that are explicitly confirmed
- Keep operational HSM monitoring separate from cryptographic HSM key operations
- If vendor documentation confirms only monitoring endpoints, implement only monitoring endpoints

## Non-Functional Requirements
- Stateless service; horizontally scalable  
- Structured logging (zap, key-value pairs)  
- Context propagation and timeouts throughout  
- No secret leakage via logs, errors, or HTTP responses  
- In-memory DEK cache with TTL and on-evict zeroization  
- All config via environment variables — no hardcoded values  

## Adding a New Database Driver
1. Create `internal/repository/<driver>/record_repo.go` + `audit_repo.go`  
2. Implement `port.RecordRepository` and `port.AuditRepository`  
3. Add a case in `internal/repository/factory.go`  
4. Add a migration in `migrations/<driver>/001_init.sql`  
5. Add a `migrate-<driver>` target in `Makefile`

### CERTEX HSM ES Verified API Surface

According to the currently available CERTEX HSM ES vendor API document, the verified API surface is REST-based and includes operational/service endpoints such as:

- `POST /clear`
- `POST /info`
- `POST /infocluster`
- `POST /infogen`
- `POST /infolog`
- `POST /findkey/{name}`
- `POST /logcount`
- `POST /date`
- `POST /battery`
- `POST /updatetime`

These endpoints are suitable for:
- health checks
- readiness checks
- node diagnostics
- cluster diagnostics
- synchronization status
- battery state monitoring
- time/NTP monitoring
- key lookup by name

These endpoints are **not sufficient evidence** of payload-encryption or DEK wrap/unwrap operations.

Therefore:
- use CERTEX HSM ES REST API for operational monitoring where applicable
- do not implement DEK wrap/unwrap through guessed REST endpoints
- keep cryptographic HSM operations behind an interface until the vendor provides verified crypto operation documentation

### Health Endpoint and HSM Monitoring

`GET /health` is the single unified probe. It performs all checks on every call with a 5-second context timeout.

Response structure:
```json
{
  "status": "ok | error",
  "version": "...",
  "time": "RFC3339",
  "uptime": "3h14m5s",
  "service":  { "env", "hsm_mode", "db_driver", "hostname" },
  "runtime":  { "go_version", "goroutines", "cpus" },
  "memory":   { "alloc_mb", "total_alloc_mb", "sys_mb", "heap_alloc_mb", "heap_sys_mb", "gc_cycles" },
  "components": {
    "db":  { "status", "driver", "latency_ms", "detail" },
    "hsm": {
      "status", "mode", "latency_ms",
      "node":       { all FKeyXxx counters, id, key_count, tasks_queue, tasks_queue_net },
      "sync":       { sync_process, sync_time_ms, percent_sync_key, ... },
      "cluster":    [ { id, key_count }, ... ],
      "log_count":  { db_total, deleted, active, in_memory },
      "battery":    { need_replace, voltage_millivolts },
      "node_time":  "HSM local time string",
      "ntp_status": "200 OK | 506 Cannot talk to daemon | stub",
      "active_keys": [],
      "errors":     [ "non-fatal telemetry errors" ]
    },
    "dek_cache": { "status", "items", "ttl" }
  }
}
```

`status` at the top level is `"ok"` only when `db.status == "ok"` AND `hsm.status == "ok"`. HSM telemetry collection errors (battery, date, etc.) populate `hsm.errors[]` and do **not** set `hsm.status = "error"`.

The `health.Collector` is constructed in `main.go` and passed to the router:
```go
collector := health.New(cfg, repos.Pinger, hsmClient, dekCache)
```
`hsmClient` satisfies `port.HSMMonitor` because it is an `hsm.FullClient`.

### CERTEX REST Adapter Rules

`internal/hsm/certex_rest.go` implements `hsm.FullClient` (`port.HSM` + `port.HSMMonitor`).

Adapter rules:
- Use HTTPS only
- Use Basic HTTP authorization (`req.SetBasicAuth(username, password)`) — never log credentials
- Use POST for all vendor endpoints; default client timeout 10s
- Parse vendor JSON responses into private DTOs (struct names prefixed `certex`)
- Map vendor field names (`FKeyGenTotal`, `tasksQueue`, etc.) into port types before returning
- Keep all vendor-specific DTOs inside the `hsm` package — do not leak them into `domain/` or `health/`

Implemented `port.HSMMonitor` methods (all match actual signatures):
```go
Ping(ctx context.Context) error
NodeInfo(ctx context.Context) (port.HSMNodeInfo, port.HSMSyncInfo, error)   // POST /info
ClusterInfo(ctx context.Context) ([]port.HSMClusterNode, error)              // POST /infocluster
LogCount(ctx context.Context) (port.HSMLogCount, error)                      // POST /logcount
Date(ctx context.Context) (string, error)                                    // POST /date  (returns raw vendor string)
Battery(ctx context.Context) (port.HSMBattery, error)                        // POST /battery
NTPStatus(ctx context.Context) (string, error)                               // POST /updatetime
ActiveKeys(ctx context.Context) ([]string, error)                            // POST /infogen
```

`port.HSM` crypto methods (`WrapDEK`, `UnwrapDEK`, `CurrentKeyVersion`) return `not implemented` error pending verified vendor crypto documentation. Do not fabricate these.

Verified vendor endpoints NOT yet wired into a port method (available if needed):
- `POST /infolog` — counts state-log records (db + in-memory)
- `POST /findkey/{name}` — key lookup by name
- `POST /clear` — clear statistics counters

### Cryptographic Policy (GOST 34.12-2015 Mandatory)

DataVault production builds targeting regulated environments must use GOST 34.12-2015 (Kuznyechik) as the primary symmetric algorithm.

AES must not be used in production builds when GOST mode is enabled.

Cryptographic requirements:

- Symmetric cipher: GOST 34.12-2015 (Kuznyechik)
- Key size: 256 bits
- Mode: authenticated mode only
- Integrity: mandatory authentication tag
- Hash function: GOST 34.11-2012 (Streebog)
- Signature (if used): GOST 34.10-2012/2015

### HSM Integration Rules (PKCS#11 + GOST)

For CERTEX HSM production integration:

- Use PKCS#11 interface only.
- KEK must be generated inside HSM.
- KEK must be non-extractable.
- KEK must allow wrap/unwrap operations.
- DEK must never be extractable outside the secure boundary.

Use GOST-compatible mechanisms exposed by HSM, for example:
- CKM_GOST28147_KEY_WRAP (if supported)
- CKM_GOST28147_KEY_WRAP_PAD (if supported)
- Vendor-defined GOST wrap mechanisms

Do not assume AES-based wrap mechanisms in GOST mode.

If a PKCS#11 mechanism is not confirmed by vendor documentation, do not implement it.

### Envelope Encryption Model (GOST Mode)

In GOST mode:

1. Generate DEK using HSM or secure RNG compliant with GOST.
2. Encrypt payload locally using GOST 34.12-2015 in authenticated mode.
3. Wrap DEK using HSM KEK.
4. Store:
   - ciphertext
   - wrapped DEK
   - nonce/IV
   - authentication tag
   - algorithm identifier

HSM must not be used for bulk payload encryption.
HSM must only protect KEK and perform key wrap/unwrap.

### Algorithm Selection Rules

DataVault must support algorithm selection via configuration.

Example:

DATAVAULT_CRYPTO_MODE=gost

Supported values:
- gost
- aes (development only unless explicitly approved)

If DATAVAULT_CRYPTO_MODE=gost:
- All symmetric encryption must use GOST 34.12-2015.
- All hashing must use GOST 34.11-2012.
- All key wrapping must use GOST-compatible mechanisms.

Do not silently fallback from GOST to AES.
Fail fast if GOST mechanisms are unavailable.

### Coding Rules for GOST

- Do not implement GOST cipher manually.
- Use only:
  - HSM cryptographic primitives, or
  - vetted and approved cryptographic libraries.

- Never reimplement Kuznyechik.
- Never reimplement Streebog.
- Never mix AES and GOST in the same encrypted dataset.
- Store algorithm identifier with each encrypted record.

Each encrypted record must include:
- alg
- key_version
- iv
- tag
- wrapped_dek

### Database Schema Requirements (GOST Mode)

DV_SECURE_DATA must include:

- ALG               (e.g. "GOST_34_12_2015")
- KEY_VERSION
- DATA_ENC
- DEK_WRAPPED
- NONCE
- AUTH_TAG

Algorithm must be stored explicitly to support future migration.

### Security Requirements (Regulated Environment)

In GOST production mode:

- All cryptographic operations must comply with ST RK 1073-2007 requirements.
- HSM must be certified at required security level.
- KEK must be generated inside HSM.
- KEK must not be exportable.
- M-of-N split control should be supported for master key management.

No mock HSM allowed in production.
No software-only KEK allowed in production.

### Error Handling Rules

If:
- GOST mechanism is unavailable,
- HSM does not support required wrap mechanism,
- HSM session cannot be established,

The service must fail startup.

Do not silently downgrade to AES.
Do not automatically switch to mock mode.
