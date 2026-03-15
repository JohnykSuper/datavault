# Changelog

All notable changes to DataVault are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).  
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Changed
- `/health` now returns `version` and `time` (RFC 3339 UTC) in addition to `status`
- `/ready` now checks both DB and HSM connectivity and returns a `components` map  
  with per-component `status`/`detail`; timeout increased from 3 s to 5 s
- `GET /health` and `GET /ready` responses are JSON objects (was plain `{"status":"…"}`)
- Database connection pool parameters are now fully configurable via env vars  
  (`DATAVAULT_DB_MAX_CONNS`, `DATAVAULT_DB_MIN_CONNS`, `DATAVAULT_DB_CONN_MAX_LIFETIME`,  
  `DATAVAULT_DB_CONN_MAX_IDLE_TIME`, `DATAVAULT_DB_HEALTH_CHECK_PERIOD`)
- Build embeds git tag as version string via ldflags  
  (`-X github.com/your-org/datavault/internal/version.Version=<tag>`)

### Added
- `internal/version` package — single source of truth for the build version
- `port.HSM.Ping(ctx)` method — HSM liveness used by `/ready`

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
