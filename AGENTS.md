# DataVault — System Instructions (Build Specification)

> **Note on instruction files**
> - `AGENTS.md` (this file) is the **original build specification** — the requirements used to scaffold the project. It is the authoritative source of *what* DataVault must do.
> - `.github/copilot-instructions.md` is the **live developer context** for GitHub Copilot — file paths, naming conventions, and workflows reflecting the *current* codebase state. Keep it up-to-date as the code evolves.
> - When there is a conflict between the two, `.github/copilot-instructions.md` takes precedence for implementation details.

Build a production-grade Go service called **DataVault**.

DataVault is a secure data encryption and key management service for a certification authority environment.

## Core Principles

- Use envelope encryption.
- Encrypt payload locally using AES-256-GCM.
- Use HSM only for KEK storage and DEK wrap/unwrap.
- Never use HSM as a bulk data encryptor.
- Support Oracle, MSSQL, PostgreSQL via separate repository adapters.
- Use explicit SQL (no mandatory ORM).
- Follow clean architecture.
- Keep business logic out of HTTP handlers.
- Use structured logging.
- Do not log plaintext, DEK, or search keys.
- Support search via HMAC-SHA256 tokens.
- Implement versioned key model.
- Service must be stateless except in-memory DEK cache.

## Required Features

- Encrypt payload
- Decrypt payload
- Search by token
- Rewrap DEK
- Unified health endpoint (`GET /health`)
- Audit logging
- Config via environment variables
- In-memory DEK cache with TTL

## Database Layer

Provide separate implementations for:
- PostgreSQL (pgx v5, `$1` placeholders)
- MSSQL (go-mssqldb, `@p1` placeholders)
- Oracle (go-ora pure-Go driver, `:1` placeholders)

Do not mix SQL dialects in one repository.

## Security Constraints

- No custom crypto algorithms.
- Use AES-256-GCM only.
- Use AAD.
- Use HMAC-SHA256 for search tokens.
- Zeroize temporary keys where possible.
- Do not store plaintext in logs.
- Do not store DEK outside memory.

## Output Requirements

When generating code:
- Provide full files.
- Include correct imports.
- Indicate file paths.
- Mark TODOs for HSM integration clearly.
- Provide separate migrations for each database.
