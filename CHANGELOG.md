# Changelog

All notable changes to DataVault are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).  
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- `internal/version` package — single source of truth for the build version
- `port.HSMMonitor` interface — `NodeInfo`, `ClusterInfo`, `LogCount`, `Date`, `Battery`, `NTPStatus`, `ActiveKeys`; matching data types: `HSMNodeInfo`, `HSMSyncInfo`, `HSMClusterNode`, `HSMBattery`, `HSMLogCount`
- `internal/hsm/certex_rest.go` — CERTEX HSM ES REST adapter (`DATAVAULT_HSM_MODE=certex`); all 9 documented monitoring endpoints fully implemented; crypto ops return `not implemented` pending verified vendor documentation
- `hsm.FullClient` composite interface — embeds `port.HSM` + `port.HSMMonitor`; returned by `hsm.New()`
- `internal/health` package — `Collector` type gathers DB latency, all HSM telemetry calls, runtime metrics, and DEK cache stats in one shot
- `DATAVAULT_HSM_URL`, `DATAVAULT_HSM_USER`, `DATAVAULT_HSM_PASS` environment variables for certex mode
- `cache.DEKCache.ItemCount()` and `CacheTTL()` methods for health reporting
- Database connection pool parameters configurable via env vars  
  `DATAVAULT_DB_MAX_CONNS`, `DATAVAULT_DB_MIN_CONNS`, `DATAVAULT_DB_CONN_MAX_LIFETIME`,  
  `DATAVAULT_DB_CONN_MAX_IDLE_TIME`, `DATAVAULT_DB_HEALTH_CHECK_PERIOD`
- `port.HSM.Ping(ctx)` method on all HSM adapters
- Build embeds git tag as version string via ldflags  
  (`-X github.com/your-org/datavault/internal/version.Version=<tag>`)

### Changed
- **`GET /health` is the single unified probe** — `/ready` removed; `/health` returns  
  `200 ok` when all components (DB + HSM) are healthy, `503 error` when any component fails
- `/health` response expanded to include: `version`, `time`, `uptime`, `service`, `runtime`,  
  `memory`, and `components` (db, hsm with full telemetry, dek_cache)
- HSM telemetry failures populate `components.hsm.errors[]` and do NOT mark HSM as down
- `hsm.New()` returns `hsm.FullClient` instead of `port.HSM`; adds `certex` to supported modes
- `DATAVAULT_HSM_MODE=stub` error message updated to list all valid production modes

### Removed
- `GET /ready` endpoint (functionality merged into `GET /health`)

---

## [v0.1.0] — 2026-03-15

Initial release of DataVault — a production-grade Go service for secure data
encryption and key management in a certification-authority environment.

### Added

#### Core
- Envelope encryption model: AES-256-GCM (DEK) + HSM-managed KEK
- RFC 3394 AES Key Wrap for DEK protection
- HMAC-SHA256 search tokens (plaintext never stored)
- Versioned key model with per-tenant KEK isolation
- In-memory DEK cache with configurable TTL and on-evict zeroization

#### API
- `POST /v1/encrypt` — encrypt payload and index search tokens
- `POST /v1/decrypt` — decrypt by record ID (JSON body, no secrets in URL)
- `GET  /v1/search`  — look up record IDs by HMAC search token
- `POST /v1/rewrap-dek` — re-wrap DEK under a new key version
- `GET  /health` — liveness probe
- `GET  /ready`  — readiness probe with DB ping (503 on failure)
- Bearer token authentication with SHA-256 + constant-time comparison

#### Database support
- PostgreSQL via `pgx` v5 (`$1` placeholders)
- Microsoft SQL Server via `go-mssqldb` (`@p1` placeholders)
- Oracle via `go-ora` pure-Go driver (`:1` placeholders, no CGO)
- Schema migrations for all three dialects (`migrations/`)

#### HSM
- In-process stub HSM for development/testing (`DATAVAULT_HSM_MODE=stub`)
- Runtime guard: stub mode is fatal when `DATAVAULT_ENV=prod`
- PKCS#11 adapter skeleton (`internal/hsm/pkcs11.go`) — implementation pending

#### Infrastructure
- Multi-stage Dockerfile (golang:1.23-alpine builder + alpine production image)
- Docker Compose stack: PostgreSQL + migrate + DataVault + pgAdmin
- `docker-compose.override.yml` for local dev extras
- `Makefile` with build, test, run, migrate, and Docker targets

#### Observability
- Structured logging via `go.uber.org/zap` (key–value pairs, no sensitive data)
- Audit event recording for every encrypt/decrypt/search/rewrap operation

#### Configuration
- All runtime config via environment variables with `DATAVAULT_` prefix
- `.env.example` template provided; `.env` excluded from VCS via `.gitignore`

### Security notes
- No hardcoded secrets, PINs, DSNs, or internal IPs in source code
- `DATAVAULT_SEARCH_KEY` and `DATAVAULT_API_KEY` are never logged
- Temporary DEK bytes are zeroized after use (`crypto.Zeroize`)

---

[v0.1.0]: https://github.com/your-org/datavault/releases/tag/v0.1.0
