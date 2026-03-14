# DataVault Naming Convention

## Product Name
- DataVault

## Technical Name
- datavault

## Repository
- datavault

## Go Module
- github.com/<org>/datavault

## Package Naming
- lowercase only

Examples:
- api
- config
- crypto
- hsm
- repository
- postgres
- mssql
- oracle

## File Naming
- snake_case

Examples:
- main.go
- aes_gcm.go
- search_token.go
- health_handler.go

## Go Types
- PascalCase

Examples:
- Service
- Config
- SecureDataRecord
- EncryptRequest
- EncryptResponse
- HSMClient
- SecureDataRepository

## Go Variables
- camelCase

Examples:
- entityType
- entityID
- kekID
- wrappedDEK
- ciphertext
- plaintext
- requestID

## JSON Fields
- camelCase

Examples:
- entityType
- entityId
- plaintextBase64
- ciphertextBase64
- wrappedDek
- kekId
- keyVersion

## API Endpoints
- lowercase paths
- kebab-case if needed

Examples:
- /v1/encrypt
- /v1/decrypt
- /v1/search
- /v1/rewrap-dek
- /health
- /ready
- /metrics

## Environment Variables
- uppercase with DATAVAULT_ prefix

Examples:
- DATAVAULT_APP_PORT
- DATAVAULT_DB_DRIVER
- DATAVAULT_DB_DSN
- DATAVAULT_HSM_MODE
- DATAVAULT_DEK_CACHE_TTL

## Database Objects
- UPPER_SNAKE_CASE
- prefix all DataVault tables with DV_

Tables:
- DV_SECURE_DATA
- DV_SEARCH_TOKEN
- DV_AUDIT_LOG
- DV_KEY_ROTATION_JOB

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

## DB Constraint Naming
Indexes:
- IDX_<TABLE>_<FIELDS>

Unique constraints:
- UQ_<TABLE>_<FIELDS>

Foreign keys:
- FK_<CHILD>_<PARENT>

Check constraints:
- CHK_<TABLE>_<NAME>

## Metrics Naming
- snake_case
- prefix all metrics with datavault_

Examples:
- datavault_http_requests_total
- datavault_hsm_requests_total
- datavault_encrypt_operations_total
- datavault_dek_cache_hits_total

## Audit Operation Names
- encrypt
- decrypt
- search
- rewrap_dek
- rotate_kek
- rotate_dek

## Branch Naming
- feature/<name>
- bugfix/<name>
- hotfix/<name>

Examples:
- feature/encrypt-api
- feature/postgres-repository
- bugfix/decrypt-aad-check

## Version Tags
- v0.1.0
- v0.2.0
- v1.0.0
