package service

import (
	"context"
	"fmt"

	"github.com/your-org/datavault/internal/crypto"
)

// DecryptRequest carries the record locator and caller metadata.
type DecryptRequest struct {
	TenantID  string
	RecordID  string
	Actor     string
	IPAddress string
}

// DecryptResponse returns the plaintext payload.
type DecryptResponse struct {
	Plaintext []byte
}

// Decrypt performs envelope decryption:
//  1. Load the record from the repository.
//  2. Unwrap the DEK via HSM (cache first).
//  3. Decrypt AES-256-GCM ciphertext.
//  4. Write audit log.
func (s *Service) Decrypt(ctx context.Context, req DecryptRequest) (*DecryptResponse, error) {
	record, err := s.records.FindByID(ctx, req.TenantID, req.RecordID)
	if err != nil {
		s.writeAudit(ctx, req.TenantID, req.RecordID, "decrypt", req.Actor, req.IPAddress, "failure", "record not found")
		return nil, fmt.Errorf("find record: %w", err)
	}

	dek, err := s.unwrapDEK(ctx, req.TenantID, record.KeyVersion, record.WrappedDEK)
	if err != nil {
		s.writeAudit(ctx, req.TenantID, req.RecordID, "decrypt", req.Actor, req.IPAddress, "failure", "DEK unwrap error")
		return nil, fmt.Errorf("unwrap DEK: %w", err)
	}
	defer crypto.Zeroize(dek)

	plaintext, err := crypto.DecryptAESGCM(dek, record.Nonce, record.Ciphertext, record.AAD)
	if err != nil {
		s.writeAudit(ctx, req.TenantID, req.RecordID, "decrypt", req.Actor, req.IPAddress, "failure", "AEAD verify failed")
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	s.writeAudit(ctx, req.TenantID, req.RecordID, "decrypt", req.Actor, req.IPAddress, "success", "")

	return &DecryptResponse{Plaintext: plaintext}, nil
}

// unwrapDEK returns the plaintext DEK, checking the cache before calling HSM.
// NOTE: caller must zeroize the returned slice.
func (s *Service) unwrapDEK(ctx context.Context, tenantID string, keyVer int, wrappedDEK []byte) ([]byte, error) {
	if cached, ok := s.cache.Get(tenantID, keyVer); ok {
		out := make([]byte, len(cached))
		copy(out, cached)
		return out, nil
	}

	plainDEK, err := s.hsm.UnwrapDEK(ctx, tenantID, keyVer, wrappedDEK)
	if err != nil {
		return nil, err
	}

	// Populate cache.
	cached := make([]byte, len(plainDEK))
	copy(cached, plainDEK)
	s.cache.Set(tenantID, keyVer, cached)

	return plainDEK, nil
}
