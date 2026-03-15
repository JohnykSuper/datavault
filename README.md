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
- In-memory DEK cache
- Structured audit logging
- Health and readiness endpoints

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

POST /v1/encrypt  
POST /v1/decrypt  
GET /v1/search  
POST /v1/rewrap-dek  
GET /health  
GET /ready  

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

Configured via environment variables:

- DB_DRIVER
- DB_DSN
- HSM_MODE
- DEK_CACHE_TTL
- LOG_LEVEL
- REQUEST_TIMEOUT
- APP_PORT

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
