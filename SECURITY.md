# DataVault Security Model

## Overview

DataVault is a centralized encryption and key management service designed to protect sensitive data stored in enterprise databases.

This document defines:
- security goals
- trust model
- threat model
- protected assets
- attack surfaces
- security controls
- operational security requirements

---

# 1. Security Goals

DataVault must provide:

- confidentiality of protected data
- integrity of encrypted payloads
- protection of master keys via HSM
- resistance to ciphertext substitution
- reduced exposure of plaintext in application flows
- safe support for exact-match search via tokenization
- auditable security-sensitive operations

---

# 2. Primary Security Objectives

## 2.1 Confidentiality
Sensitive data must not be recoverable from database contents alone.

## 2.2 Integrity
Encrypted payloads must detect tampering.

## 2.3 Key Separation
Data encryption keys must be separated from master keys.

## 2.4 Limited Trust in Database
The database must not be treated as a trusted cryptographic boundary.

## 2.5 Controlled Key Access
Only DataVault service logic may request key operations through HSM integration.

---

# 3. Trust Model

## 3.1 Trusted Components
The following components are part of the trusted computing base:

- HSM
- DataVault service process
- secure deployment environment
- approved runtime configuration
- trusted secret delivery mechanism for service bootstrap

## 3.2 Partially Trusted Components
The following components are operationally required but not fully trusted for confidentiality:

- database
- application network
- infrastructure logs
- administrators with database access

## 3.3 Untrusted or Adversarial Context
The following must be assumed potentially observable or hostile:

- raw database dumps
- backup copies
- intercepted SQL-level storage access
- compromised read-only DB accounts
- accidental log exposure
- unauthorized infrastructure observers

---

# 4. Root of Trust

The HSM is the root of trust for key protection.

HSM responsibilities:
- hold KEK securely
- wrap DEK
- unwrap DEK
- provide search key material or equivalent protected access path
- support key version lifecycle

The HSM must not be used as a bulk data encryptor for application payloads.

---

# 5. Protected Assets

## 5.1 Critical Assets
- KEK
- search key
- plaintext payload
- decrypted DEK in memory
- HSM credentials and session context

## 5.2 High-Value Assets
- wrapped DEK
- ciphertext
- audit trail
- metadata linking records to business entities
- rotation state

## 5.3 Operational Security Assets
- service configuration
- deployment manifests
- TLS configuration
- monitoring and alerting pipelines
- CI/CD secrets

---

# 6. Data Classification Inside DataVault

## 6.1 Secret
Must never be logged or stored outside protected memory:
- plaintext
- DEK
- search key
- HSM PIN / token credentials

## 6.2 Sensitive
Must be tightly controlled:
- wrapped DEK
- ciphertext
- entity identifiers
- request correlation IDs in regulated contexts

## 6.3 Internal
Allowed in controlled observability channels:
- operation name
- duration
- result code
- DB driver type
- service version

---

# 7. Threat Model

## 7.1 Threat: Database Dump Theft
An attacker obtains a full dump of application tables.

### Risk
Exposure of sensitive user data.

### Control
- ciphertext stored instead of plaintext
- DEK stored only wrapped
- KEK stored only in HSM
- search tokens computed via HMAC, not reversible

### Residual Risk
Metadata exposure remains possible:
- entity type
- entity identifier
- access patterns
- approximate payload size

---

## 7.2 Threat: Ciphertext Substitution
An attacker replaces ciphertext from one record with ciphertext from another.

### Risk
Unauthorized data confusion or replay.

### Control
Use mandatory AAD including:
- entity_type
- entity_id
- optional version metadata

This binds ciphertext to the record context.

---

## 7.3 Threat: Compromised DB Read Access
An attacker gains read-only access to DB.

### Risk
Access to ciphertext, wrapped keys, search tokens.

### Control
- no plaintext in DB
- no KEK in DB
- tokenization instead of direct searchable plaintext
- HSM-protected KEK lifecycle

---

## 7.4 Threat: Compromised Application Logs
Logs are leaked or misconfigured.

### Risk
Sensitive information disclosure.

### Control
Never log:
- plaintext
- DEK
- search keys
- full ciphertext
- full wrapped DEK
- sensitive request bodies

Use structured logging with controlled fields only.

---

## 7.5 Threat: HSM Abuse via Application
An attacker attempts to use DataVault as a proxy to misuse HSM operations.

### Risk
Improper key usage or large-scale unauthorized unwrap activity.

### Control
- strict service API boundaries
- no raw HSM passthrough endpoints
- domain-level validation
- audit every security-sensitive operation
- rate limiting / anomaly detection as future extension

---

## 7.6 Threat: Memory Disclosure
An attacker obtains process memory or dump.

### Risk
Exposure of plaintext and DEK in RAM.

### Control
- minimize plaintext lifetime
- minimize DEK lifetime
- zeroize buffers where practical
- avoid persistent caches of plaintext
- keep DEK cache TTL short
- disable verbose diagnostics in production

### Residual Risk
A fully compromised host can still expose runtime memory.

---

## 7.7 Threat: Replay or Duplicate Requests
Client repeats an operation intentionally or accidentally.

### Risk
Unexpected repeated encryption or rotation.

### Control
- request tracing
- audit trail
- optional idempotency extension in future
- explicit operation boundaries

---

## 7.8 Threat: Search Token Abuse
An attacker uses search endpoint to probe presence of known values.

### Risk
Existence disclosure by exact-match probing.

### Control
- authenticate and authorize clients externally
- audit search operations
- consider rate limiting
- limit searchable fields
- avoid exposing unrestricted search publicly

### Residual Risk
Exact-match token search inherently leaks existence under authorized use.

---

## 7.9 Threat: Misconfiguration
Unsafe startup config, dev-mode HSM in production, weak timeouts, or wrong database mode.

### Risk
Security controls bypassed by operational error.

### Control
- strict startup validation
- explicit environment modes
- fail-fast configuration loading
- production guardrails
- configuration review checklist

---

# 8. Attack Surfaces

Primary attack surfaces:

- HTTP API
- HSM adapter boundary
- database connections
- configuration and secret injection path
- logs and telemetry
- CI/CD pipeline
- container orchestration layer

Each surface must be treated as security-relevant.

---

# 9. Security Controls

## 9.1 Cryptographic Controls
- AES-256-GCM only for payload encryption
- HMAC-SHA256 for search token generation
- envelope encryption only
- HSM-protected KEK
- AAD required
- no custom crypto algorithms

## 9.2 Key Management Controls
- KEK never stored in DB
- DEK only stored wrapped
- versioned key identifiers
- support for KEK rotation
- support for DEK rotation where required

## 9.3 Application Controls
- request validation
- explicit DTO boundaries
- strict repository interfaces
- service layer controls around HSM usage
- no direct HSM exposure to callers

## 9.4 Operational Controls
- structured logging
- audit logging
- environment-based config validation
- readiness checks
- secret injection control
- production-safe defaults

## 9.5 Access Controls
DataVault assumes authentication and service-to-service authorization are enforced by surrounding infrastructure or gateway layer.

Recommended:
- mTLS or trusted internal network boundary
- service account identity
- per-client authorization policy
- restricted access to search and rewrap operations

---

# 10. Security Decisions

## 10.1 Why HSM Is Not Used for Bulk Encryption
Using HSM for every payload encryption operation would:
- reduce performance
- create bottlenecks
- increase latency
- provide no practical benefit for bulk symmetric payload encryption

Therefore:
- HSM protects KEK
- application encrypts data locally with DEK

## 10.2 Why Search Uses HMAC Tokens
Direct search over encrypted fields is impractical for exact-match lookup at scale.

Therefore:
- normalized field value
- HMAC-SHA256 with protected search key
- indexed token storage

This enables exact-match lookup with acceptable leakage profile.

## 10.3 Why Database Is Not Trusted
Database administrators, dumps, replicas, and backups may expose stored content.

Therefore:
- plaintext must not be stored
- wrapped DEK must not be sufficient to decrypt data without HSM

---

# 11. Residual Risks

The following risks remain even with correct implementation:

- metadata leakage
- record existence leakage via token search
- runtime memory exposure on compromised host
- abuse by fully privileged service operator
- misuse through surrounding infrastructure if access controls are weak

These residual risks must be managed operationally.

---

# 12. Non-Goals

DataVault does not aim to provide:

- full identity and access management
- full PKI lifecycle
- general-purpose secrets vault
- client-side encryption SDK for arbitrary environments
- protection against a fully compromised production host
- protection against malicious code already running inside the same trusted process

---

# 13. Secure Development Requirements

## Required
- code review for crypto-sensitive changes
- no custom algorithm design
- dependency review
- unit tests for crypto primitives
- test coverage for normalization and token generation
- repository integration tests per supported DB
- explicit review for log safety

## Forbidden
- logging plaintext
- logging DEK or search key
- storing secrets in source code
- production use of mock HSM
- silent fallback from real HSM to mock mode
- security-relevant TODOs left unresolved for production release

---

# 14. Secure Operations Requirements

## Deployment
- run with least privilege
- isolate production secrets
- restrict shell access
- disable debug endpoints in production unless explicitly approved
- use TLS within service perimeter where required

## Secrets
- do not store HSM credentials in repository
- rotate service credentials according to policy
- control access to CI/CD secrets
- separate dev/test/prod credentials

## Monitoring
- monitor HSM latency and failures
- monitor DB failures
- monitor unusual search volume
- monitor repeated decrypt failures
- alert on readiness degradation

---

# 15. Recommended Future Hardening

- mTLS between clients and DataVault
- per-operation authorization policy
- dedicated admin API separation
- rate limiting on search and decrypt endpoints
- tamper-evident audit storage
- hardware-backed host protections
- envelope re-encryption job framework
- split-duty operational controls for key rotation

---

# 16. Summary

DataVault security is built on the following principles:

- HSM as root of trust
- envelope encryption
- application-layer cryptographic enforcement
- limited trust in database and infrastructure
- minimal exposure of plaintext and keys
- auditable security-sensitive behavior

The design assumes that confidentiality must survive database compromise and that cryptographic control must remain outside database trust boundaries.
