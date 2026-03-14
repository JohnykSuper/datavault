# DataVault — Full Engineering Instructions

## Overview

DataVault is a centralized encryption and key management service designed for secure storage of sensitive data in enterprise environments.

The system must support:
- Envelope encryption
- HSM-based KEK management
- Search tokenization
- Multi-database support (Oracle, MSSQL, PostgreSQL)
- Production-ready architecture

---

## Cryptographic Model

### Envelope Encryption

KEK:
- Stored in HSM
- Used only for wrapping/unwrapping DEK

DEK:
- Generated per logical object
- Used for AES-256-GCM encryption
- Stored in DB only in wrapped form

---

## Algorithms

- AES-256-GCM
- Nonce: 12 bytes
- Auth tag: 16 bytes
- HMAC-SHA256 for search tokens
- Mandatory AAD usage

---

## Architecture

Follow Clean Architecture.

Layers:
- Domain
- Crypto
- HSM Adapter
- Repository
- Service
- API
- Config
- Audit
- Metrics

---

## Database Requirements

Support:

PostgreSQL:
- BYTEA
- TIMESTAMPTZ
- RETURNING id

MSSQL:
- VARBINARY(MAX)
- DATETIME2
- IDENTITY

Oracle:
- BLOB
- RAW
- SYSTIMESTAMP
- Sequence/identity handling

Each DB must have:
- Separate repository implementation
- Separate SQL migrations
- Separate placeholder syntax handling

---

## Required Tables

Main table:
- id
- entity_type
- entity_id
- data_enc
- dek_wrapped
- nonce
- auth_tag
- alg
- kek_id
- key_version
- created_at
- updated_at

Search token table:
- id
- record_id
- field_name
- token

Audit table:
- id
- operation
- entity_type
- entity_id
- result
- error_code
- duration_ms
- created_at

---

## Service Endpoints

- POST /v1/encrypt
- POST /v1/decrypt
- GET /v1/search
- POST /v1/rewrap-dek
- GET /health
- GET /ready

---

## Non-Functional Requirements

- Stateless
- Horizontal scaling
- Structured logging
- Context timeouts
- No secret leakage
- In-memory DEK cache
- Config via environment variables
- Separate dev mock HSM
- Production PKCS#11 HSM adapter

---

## Coding Rules

- Do not mix SQL dialects.
- Do not log sensitive data.
- Do not implement insecure mock crypto for production.
- Prefer explicit SQL over ORM.
- Keep interfaces clean.
- Write unit tests for crypto primitives.
- Use Go standard library where possible.
- Avoid unnecessary external dependencies.

---

## Order of Implementation

1. Domain models
2. Crypto primitives
3. HSM interface + mock
4. Service layer
5. Repository interfaces
6. Postgres adapter
7. MSSQL adapter
8. Oracle adapter
9. HTTP layer
10. Migrations
11. Tests
12. Metrics and audit

---

## Final Goal

Deliver a production-ready encryption service suitable for enterprise certification authority infrastructure.
