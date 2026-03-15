# DataVault

DataVault is a secure data encryption and key management service.

It provides centralized encryption, decryption, and search-token capabilities using envelope encryption and HSM-based master key protection.

---

## Core Features

- AES-256-GCM encryption
- Envelope encryption model
- HSM-based KEK management
- Per-record DEK
- Search token support via HMAC-SHA256
- Multi-database support:
  - Oracle
  - Microsoft SQL Server
  - PostgreSQL
- Versioned key management
- Rewrap support for KEK rotation
- In-memory DEK cache with TTL and on-evict zeroization
- Structured audit logging
- Unified health endpoint with full HSM telemetry (`GET /health`)

---

## Architecture

DataVault follows clean architecture principles.

Layers:
- Domain
- Crypto
- HSM Adapter
- Repository (Oracle / MSSQL / PostgreSQL)
- Service
- API
- Audit
- Metrics

---

## Cryptographic Model

### KEK
Stored in HSM.
Used only for wrapping/unwrapping DEK.

### DEK
Generated per object.
Used for AES-256-GCM encryption.
Stored only in wrapped form.

---

## Supported Databases

### PostgreSQL
- BYTEA
- TIMESTAMPTZ

### MSSQL
- VARBINARY(MAX)
- DATETIME2

### Oracle
- BLOB
- RAW
- SYSTIMESTAMP

---

## API

All `/v1/*` routes require `Authorization: Bearer <DATAVAULT_API_KEY>`.

```
POST /v1/encrypt
POST /v1/decrypt
GET  /v1/search
POST /v1/rewrap-dek
GET  /health        — 200 ok (all components healthy) | 503 error (any component down)
```

The `/health` response includes service info, runtime metrics, memory stats,
DB latency, and full HSM telemetry (node counters, sync state, battery, NTP status).

---

## Security Principles

- No plaintext logging
- No DEK persistence outside memory
- No HSM bulk encryption
- Mandatory AAD
- No custom crypto
- Explicit SQL, no forced ORM

---

## Configuration

Configured via environment variables (all prefixed `DATAVAULT_`):

```
DATAVAULT_APP_PORT          # HTTP listen port (default 8080)
DATAVAULT_ENV               # dev | test | prod
DATAVAULT_DB_DRIVER         # postgres | mssql | oracle
DATAVAULT_DB_DSN            # connection string (postgres)
DATAVAULT_HSM_MODE          # stub | certex | pkcs11
DATAVAULT_HSM_URL           # CERTEX HSM ES base URL (certex mode)
DATAVAULT_HSM_USER          # HSM Basic-auth username (certex mode)
DATAVAULT_HSM_PASS          # HSM Basic-auth password (certex mode)
DATAVAULT_SEARCH_KEY        # 64-char hex, 32-byte HMAC key
DATAVAULT_API_KEY           # Bearer token for /v1/* routes
DATAVAULT_DEK_CACHE_TTL     # e.g. 5m
DATAVAULT_LOG_LEVEL         # debug | info | warn | error
```

See `.env.example` for all options including DB pool settings.

---

## Deployment Model

- Stateless
- Horizontally scalable
- Container-friendly
- Supports enterprise HSM integration

---

## Future Extensions

- Chunked encryption for large objects
- gRPC transport
- Key rotation scheduler
- Multi-tenant key separation
- Event streaming
- Vault integration

---

## License

DataVault is licensed under the Apache License 2.0.

See the LICENSE file for details.

---

## Disclaimer

DataVault is provided for educational and engineering purposes.
It is not a certified security product.
Users are responsible for evaluating security suitability for their environment.
