# DataVault Architecture

## Overview

DataVault is a centralized encryption and key management service designed for secure storage of sensitive data in enterprise environments.

The system implements envelope encryption with HSM-backed key protection and supports multiple database engines:
- PostgreSQL
- Microsoft SQL Server
- Oracle

DataVault is stateless, horizontally scalable, and production-ready.

---

# 1. Design Goals

- Strong cryptographic guarantees
- High performance
- Horizontal scalability
- Multi-database support
- Clear separation of concerns
- HSM abstraction
- Secure-by-default behavior
- Explicit SQL (no mandatory ORM)
- Observability and auditability

---

# 2. High-Level Architecture

Client → HTTP API → Service Layer → Crypto Layer → HSM Adapter  
                                        ↓  
                                   Repository Layer → Database

Core layers:

- API Layer
- Application / Service Layer
- Crypto Layer
- HSM Adapter Layer
- Repository Layer
- Audit Layer
- Metrics Layer
- Configuration Layer

---

# 3. Cryptographic Model

## 3.1 Envelope Encryption

DataVault uses envelope encryption.

### KEK (Key Encryption Key)
- Stored inside HSM
- Never stored in database
- Used only to wrap/unwrap DEK

### DEK (Data Encryption Key)
- Generated per logical object
- Used for AES-256-GCM encryption
- Stored in DB only in wrapped form

---

## 3.2 Encryption Algorithm

- AES-256-GCM
- Nonce: 12 bytes
- Auth tag: 16 bytes
- Mandatory AAD

AAD includes:
- entity_type
- entity_id
- optional metadata/version

This prevents ciphertext substitution attacks.

---

## 3.3 Search Token Model

For searchable fields, DataVault uses:

- HMAC-SHA256
- Dedicated search key
- Normalized input value

Search tokens allow exact-match queries without decrypting data.

---

# 4. Layered Architecture

## 4.1 API Layer

Responsibilities:
- HTTP routing
- DTO validation
- Error mapping
- Request context management

Must not:
- Contain business logic
- Call HSM directly
- Perform encryption directly

Endpoints:
- POST /v1/encrypt
- POST /v1/decrypt
- GET /v1/search
- POST /v1/rewrap-dek
- GET /health
- GET /ready

---

## 4.2 Service Layer

Responsibilities:
- Orchestrate encryption/decryption
- Manage DEK generation
- Coordinate HSM wrap/unwrap
- Handle search token generation
- Trigger audit logging
- Enforce security rules

This is the core application layer.

---

## 4.3 Crypto Layer

Responsibilities:
- AES-256-GCM encryption
- AES-256-GCM decryption
- DEK generation
- HMAC-SHA256 search token generation
- Value normalization
- Zeroization helper

Crypto layer must not:
- Access database
- Access HTTP
- Access configuration directly

---

## 4.4 HSM Adapter Layer

Responsibilities:
- Provide abstract interface:
  - CurrentKEK()
  - WrapKey()
  - UnwrapKey()
  - SearchKey()
- Implementations:
  - Mock (dev only)
  - PKCS#11 adapter
  - Vendor SDK adapter

Business logic must depend only on the interface.

---

## 4.5 Repository Layer

Responsibilities:
- Persist encrypted records
- Persist search tokens
- Persist audit logs
- Retrieve records
- Update wrapped DEK (for rewrap)

Separate implementations for:
- PostgreSQL
- MSSQL
- Oracle

No mixed SQL dialects in same file.

---

## 4.6 Audit Layer

Responsibilities:
- Log security-sensitive operations:
  - encrypt
  - decrypt
  - search
  - rewrap_dek
  - rotate_kek
- Store metadata only
- Never store plaintext or DEK

Audit must include:
- operation
- entity_type
- entity_id
- result
- duration
- error code
- timestamp

---

## 4.7 Metrics Layer

Expose Prometheus-compatible metrics.

Key metrics:
- datavault_http_requests_total
- datavault_encrypt_operations_total
- datavault_decrypt_operations_total
- datavault_hsm_request_duration_seconds
- datavault_db_query_duration_seconds
- datavault_dek_cache_hits_total
- datavault_dek_cache_misses_total

---

## 4.8 Configuration Layer

Configuration via environment variables.

Examples:
- DATAVAULT_DB_DRIVER
- DATAVAULT_DB_DSN
- DATAVAULT_HSM_MODE
- DATAVAULT_DEK_CACHE_TTL
- DATAVAULT_REQUEST_TIMEOUT

Config validated at startup.

---

# 5. Database Design

## 5.1 Main Table: DV_SECURE_DATA

Columns:
- ID
- ENTITY_TYPE
- ENTITY_ID
- DATA_ENC
- DEK_WRAPPED
- NONCE
- AUTH_TAG
- ALG
- KEK_ID
- KEY_VERSION
- CREATED_AT
- UPDATED_AT

---

## 5.2 Search Token Table: DV_SEARCH_TOKEN

Columns:
- ID
- RECORD_ID
- FIELD_NAME
- TOKEN
- CREATED_AT

Indexed by:
- FIELD_NAME
- TOKEN

---

## 5.3 Audit Table: DV_AUDIT_LOG

Columns:
- ID
- REQUEST_ID
- OPERATION
- ENTITY_TYPE
- ENTITY_ID
- RESULT
- ERROR_CODE
- DURATION_MS
- CREATED_AT

---

# 6. Key Rotation Strategy

## 6.1 KEK Rotation

- New KEK generated in HSM
- Existing DEK rewrapped
- DATA_ENC remains unchanged

Process:
1. Load record
2. Unwrap DEK using old KEK
3. Wrap DEK using new KEK
4. Update KEK_ID and KEY_VERSION

---

## 6.2 DEK Rotation

Requires:
- Decrypt payload
- Generate new DEK
- Re-encrypt payload
- Wrap new DEK
- Update record

Used rarely.

---

# 7. Performance Model

## 7.1 DEK Cache

In-memory only.
TTL-based.
Keyed by:
- hash(kek_id + wrapped_dek)

Purpose:
- Reduce HSM calls
- Improve decrypt performance

Never persisted.

---

## 7.2 Scalability

Service is stateless:
- No local storage
- No persistent session
- Horizontal scaling supported

---

# 8. Security Principles

- No plaintext logging
- No DEK persistence outside RAM
- HSM used only for KEK operations
- No deterministic encryption for sensitive data
- Strict separation of responsibilities
- Zeroization where possible
- Context timeouts on all external calls

---

# 9. Deployment Model

- Containerized
- Kubernetes-ready
- Stateless
- Separate environments:
  - dev
  - test
  - prod

Supports:
- External HSM cluster
- External database cluster

---

# 10. Future Extensions

- Chunked encryption for large objects
- gRPC transport
- Multi-tenant key isolation
- External KMS integration
- Vault integration
- Asynchronous key rotation jobs
- Event streaming (Kafka)
- Cross-region replication

---

# 11. Trust Boundary

DataVault assumes:
- Database is not fully trusted
- Network may be observed
- Application clients are authenticated externally
- HSM is the root of trust

The HSM is the only component trusted with master key material.

---

# 12. Summary

DataVault is:

- A cryptographic enforcement layer
- A key orchestration service
- A database-agnostic encryption gateway
- A horizontally scalable security platform

It is not:
- A database replacement
- A general secret manager
- A full PKI system
- A storage engine

It is a focused encryption and key management service.
