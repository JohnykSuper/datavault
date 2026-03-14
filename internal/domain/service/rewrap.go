package service

import (
	"context"
	"fmt"

	"github.com/your-org/datavault/internal/crypto"
)

// RewrapRequest migrates all records for a tenant from an old key version to
// the current active key version.
type RewrapRequest struct {
	TenantID   string
	OldVersion int
	Actor      string
	IPAddress  string
}

// RewrapResponse reports how many records were migrated.
type RewrapResponse struct {
	Migrated int
}

// RewrapDEK re-encrypts the DEK of each record under the new KEK version
// without exposing plaintext payloads. The sequence per record is:
//  1. Unwrap the old DEK via HSM.
//  2. Re-wrap with the new KEK version.
//  3. Persist the updated wrapped DEK.
//  4. Evict the old DEK from cache.
func (s *Service) RewrapDEK(ctx context.Context, req RewrapRequest) (*RewrapResponse, error) {
	newVer, err := s.hsm.CurrentKeyVersion(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("get current key version: %w", err)
	}

	const batchSize = 100
	migrated := 0

	for offset := 0; ; offset += batchSize {
		batch, err := s.records.ListByKeyVersion(ctx, req.TenantID, req.OldVersion, batchSize, offset)
		if err != nil {
			return nil, fmt.Errorf("list records: %w", err)
		}
		if len(batch) == 0 {
			break
		}

		for _, record := range batch {
			dek, err := s.hsm.UnwrapDEK(ctx, req.TenantID, record.KeyVersion, record.WrappedDEK)
			if err != nil {
				return nil, fmt.Errorf("unwrap DEK for record %s: %w", record.ID, err)
			}

			newWrapped, err := s.hsm.WrapDEK(ctx, req.TenantID, newVer, dek)
			crypto.Zeroize(dek)
			if err != nil {
				return nil, fmt.Errorf("wrap DEK for record %s: %w", record.ID, err)
			}

			record.WrappedDEK = newWrapped
			record.KeyVersion = newVer
			if err := s.records.Update(ctx, record); err != nil {
				return nil, fmt.Errorf("update record %s: %w", record.ID, err)
			}

			s.cache.Delete(req.TenantID, req.OldVersion)
			migrated++
		}
	}

	s.writeAudit(ctx, req.TenantID, "", "rewrap", req.Actor, req.IPAddress, "success",
		fmt.Sprintf("old_version=%d new_version=%d migrated=%d", req.OldVersion, newVer, migrated))

	return &RewrapResponse{Migrated: migrated}, nil
}
