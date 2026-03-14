package service

import (
	"context"
	"crypto/rand"
	"fmt"

	"github.com/google/uuid"
	"github.com/your-org/datavault/internal/crypto"
	"github.com/your-org/datavault/internal/domain/model"
)

// EncryptRequest carries the plaintext payload and metadata for encryption.
type EncryptRequest struct {
	TenantID     string
	Plaintext    []byte
	AAD          []byte   // additional authenticated data (e.g. tenant+record IDs)
	SearchFields []string // plaintext values to index as HMAC tokens
	Actor        string
	IPAddress    string
}

// EncryptResponse returns the record ID after successful encryption.
type EncryptResponse struct {
	RecordID string
}

// Encrypt performs envelope encryption:
//  1. Resolve current key version from HSM.
//  2. Get or generate a DEK (check cache first).
//  3. Encrypt payload with AES-256-GCM using the DEK.
//  4. Wrap DEK with HSM KEK.
//  5. Persist the record and search tokens.
//  6. Write audit log.
func (s *Service) Encrypt(ctx context.Context, req EncryptRequest) (*EncryptResponse, error) {
	keyVer, err := s.hsm.CurrentKeyVersion(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("get key version: %w", err)
	}

	dek, err := s.getOrGenerateDEK(ctx, req.TenantID, keyVer)
	if err != nil {
		return nil, fmt.Errorf("get DEK: %w", err)
	}
	defer crypto.Zeroize(dek)

	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext, err := crypto.EncryptAESGCM(dek, nonce, req.Plaintext, req.AAD)
	if err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}

	wrappedDEK, err := s.hsm.WrapDEK(ctx, req.TenantID, keyVer, dek)
	if err != nil {
		return nil, fmt.Errorf("wrap DEK: %w", err)
	}

	tokens := make([]string, len(req.SearchFields))
	for i, field := range req.SearchFields {
		tokens[i] = crypto.HMACSha256Token(s.cfg.HMACKey, []byte(field))
	}

	record := &model.Record{
		ID:           uuid.New().String(),
		TenantID:     req.TenantID,
		Ciphertext:   ciphertext,
		Nonce:        nonce,
		AAD:          req.AAD,
		WrappedDEK:   wrappedDEK,
		KeyVersion:   keyVer,
		SearchTokens: tokens,
	}

	if err := s.records.Save(ctx, record); err != nil {
		return nil, fmt.Errorf("save record: %w", err)
	}

	s.writeAudit(ctx, req.TenantID, record.ID, "encrypt", req.Actor, req.IPAddress, "success", "")

	return &EncryptResponse{RecordID: record.ID}, nil
}

// getOrGenerateDEK returns a cached DEK or generates a new one.
// NOTE: caller must zeroize the returned slice.
func (s *Service) getOrGenerateDEK(ctx context.Context, tenantID string, keyVer int) ([]byte, error) {
	if dek, ok := s.cache.Get(tenantID, keyVer); ok {
		// Return a copy so the cache retains ownership of its copy.
		out := make([]byte, len(dek))
		copy(out, dek)
		return out, nil
	}

	// Generate a fresh 32-byte DEK.
	dek := make([]byte, 32)
	if _, err := rand.Read(dek); err != nil {
		return nil, err
	}

	// Cache a copy; the cache owns that copy.
	cached := make([]byte, 32)
	copy(cached, dek)
	s.cache.Set(tenantID, keyVer, cached)

	return dek, nil
}
