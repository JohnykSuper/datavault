package service

import (
	"context"
	"fmt"

	"github.com/your-org/datavault/internal/crypto"
)

// SearchRequest searches for records by a plaintext field value.
// The plaintext is converted to an HMAC token server-side; it is never logged.
type SearchRequest struct {
	TenantID   string
	FieldValue string // plaintext — will be HMAC-hashed, never persisted
	Actor      string
	IPAddress  string
}

// SearchResponse returns the matching record IDs.
type SearchResponse struct {
	RecordIDs []string
}

// Search derives an HMAC-SHA256 token from the field value and queries the
// search_tokens index. No plaintext ever reaches the database.
func (s *Service) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	token := crypto.HMACSha256Token(s.cfg.HMACKey, []byte(req.FieldValue))

	ids, err := s.records.FindBySearchToken(ctx, req.TenantID, token)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	s.writeAudit(ctx, req.TenantID, "", "search", req.Actor, req.IPAddress, "success",
		fmt.Sprintf("results=%d", len(ids)))

	return &SearchResponse{RecordIDs: ids}, nil
}
