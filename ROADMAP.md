# DataVault Roadmap

This roadmap defines the phased delivery plan for DataVault.

The goal is to move from a minimal secure prototype to a production-grade enterprise encryption platform.

---

# Versioning Strategy

DataVault follows Semantic Versioning:

- v0.x.x — pre-production, architecture stabilization
- v1.0.0 — production-ready stable release
- v1.x.x — incremental features and improvements
- v2.x.x — major architectural upgrades

---

# Phase 0 — Foundation (v0.1.0)

Goal: Working secure prototype with one database and mock HSM.

## Scope

- Clean project structure
- Config loader
- AES-256-GCM implementation
- DEK generation
- Search token implementation
- HSM interface
- Stub HSM implementation (in-process, dev/test only)
- Service layer
- HTTP API:
  - /v1/encrypt
  - /v1/decrypt
  - /v1/search
  - /health
- PostgreSQL repository implementation
- Basic migrations for PostgreSQL
- Structured logging
- In-memory DEK cache
- Unit tests for crypto primitives
- Basic integration tests (Postgres)

## Non-Goals

- Real HSM integration
- MSSQL support
- Oracle support
- Key rotation
- Metrics
- Audit persistence
- Production hardening

---

# Phase 1 — Multi-Database Support (v0.2.0)

Goal: Full multi-DB compatibility.

## Scope

- MSSQL repository implementation
- Oracle repository implementation
- Separate migrations:
  - postgres
  - mssql
  - oracle
- DB-specific integration tests
- Config validation for DB driver selection
- Performance validation across DBs

## Non-Goals

- Production HSM integration
- Advanced audit
- Metrics
- Rotation scheduler

---

# Phase 2 — Real HSM Integration (v0.3.0)

Goal: Replace stub HSM with real secure adapters.

## Scope

- CERTEX HSM ES REST adapter (monitoring endpoints: implemented ✅)
- CERTEX HSM ES crypto operations (WrapDEK/UnwrapDEK: pending verified vendor documentation)
- PKCS#11 adapter
- HSM connection pool management
- HSM timeout handling
- Retry policy
- Disable stub HSM in production mode (implemented ✅)
- Secure secret injection handling

## Validation

- Wrap/unwrap latency benchmarks
- Failure behavior validation
- Safe startup checks

---

# Phase 3 — Audit & Observability (v0.4.0)

Goal: Enterprise visibility.

## Scope

- DV_AUDIT_LOG implementation
- Audit repository
- Structured audit writer
- Prometheus metrics endpoint
- Metrics for:
  - HTTP
  - HSM
  - DB
  - Cache
- Duration tracking
- Request correlation IDs
- Readiness deep checks

---

# Phase 4 — Key Lifecycle Management (v0.5.0)

Goal: Controlled key rotation support.

## Scope

- Rewrap DEK endpoint
- KEK versioning support
- Key version metadata handling
- Rotation CLI tool or admin endpoint
- Migration job template for bulk rewrap
- Rotation audit tracking

## Optional

- Background rotation job framework
- Progressive migration support

---

# Phase 5 — Hardening (v0.6.0)

Goal: Production safety.

## Scope

- Strict config validation
- mTLS support (optional)
- Rate limiting
- Search endpoint protection
- Request size limits
- Memory handling improvements
- Panic recovery middleware
- Improved error taxonomy
- Load testing
- Stress testing
- Failure injection tests

---

# Phase 6 — Production Release (v1.0.0)

Goal: Stable enterprise release.

## Requirements

- All previous phases complete
- Production HSM validated
- All 3 databases validated
- Audit and metrics active
- Security review completed
- Threat model reviewed
- Code freeze
- Tagged release v1.0.0

Deliverables:

- Container image
- Deployment manifest
- Runbook
- Monitoring playbook
- Key rotation procedure
- Incident response procedure

---

# Phase 7 — Post-1.0 Enhancements (v1.x.x)

Future improvements:

- Chunked encryption for large BLOBs
- gRPC transport
- Async processing
- Kafka integration
- Multi-tenant key separation
- Sharded key domains
- KMS bridge integration
- External Vault integration
- Tamper-evident audit chain
- Advanced rate-limiting
- Per-client policy engine

---

# Technical Debt Policy

No security TODO may ship in v1.0.0.

Mock HSM must never be enabled in production mode.

Crypto code changes require explicit review.

---

# Release Criteria

Each version must include:

- Updated migrations
- Updated documentation
- CHANGELOG update
- Tag in repository
- Docker image built
- Deployment test passed

---

# Operational Milestones

Before v1.0.0:

- HSM latency baseline measured
- Throughput baseline measured
- Failure scenarios tested:
  - HSM down
  - DB down
  - Partial network failure
- Load test at target TPS
- Audit storage validated under load
- Log leakage review performed

---

# Long-Term Vision

DataVault evolves into:

- A centralized enterprise crypto enforcement layer
- A key orchestration platform
- A secure data tokenization gateway
- A regulated environment encryption boundary
