package service

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/your-org/datavault/internal/config"
	"github.com/your-org/datavault/internal/domain/model"
	"github.com/your-org/datavault/internal/logger"
)

// ── Mocks ────────────────────────────────────────────────────────────────────

// mockHSM is an in-process stub that uses the same RFC-3394 key wrapping
// as the real Stub, but is self-contained for testing.
type mockHSM struct {
	mu  sync.Mutex
	kek []byte // fixed 32-byte KEK for all tenants/versions
}

func newMockHSM() *mockHSM {
	kek := make([]byte, 32)
	for i := range kek {
		kek[i] = byte(i + 1)
	}
	return &mockHSM{kek: kek}
}

func (h *mockHSM) CurrentKeyVersion(_ context.Context, _ string) (int, error) { return 1, nil }

func (h *mockHSM) WrapDEK(_ context.Context, _ string, _ int, dek []byte) ([]byte, error) {
	out := make([]byte, len(dek))
	for i := range dek {
		out[i] = dek[i] ^ h.kek[i%len(h.kek)]
	}
	return out, nil
}

// Note: this simple XOR mock doesn't have an integrity check;
// we're testing service logic, not key wrapping here.
func (h *mockHSM) UnwrapDEK(_ context.Context, _ string, _ int, wrapped []byte) ([]byte, error) {
	out := make([]byte, len(wrapped))
	for i := range wrapped {
		out[i] = wrapped[i] ^ h.kek[i%len(h.kek)]
	}
	return out, nil
}

// mockCache is an unbounded in-memory DEK cache with no TTL.
type mockCache struct {
	mu    sync.RWMutex
	store map[string][]byte
}

func newMockCache() *mockCache { return &mockCache{store: make(map[string][]byte)} }

func (c *mockCache) key(tenant string, ver int) string {
	return fmt.Sprintf("%s:%d", tenant, ver)
}

func (c *mockCache) Get(tenant string, ver int) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.store[c.key(tenant, ver)]
	return v, ok
}

func (c *mockCache) Set(tenant string, ver int, dek []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[c.key(tenant, ver)] = dek
}

func (c *mockCache) Delete(tenant string, ver int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.store, c.key(tenant, ver))
}

// mockRecordRepo stores records in memory keyed by (tenantID, recordID).
type mockRecordRepo struct {
	mu      sync.RWMutex
	records map[string]*model.Record
}

func newMockRecordRepo() *mockRecordRepo {
	return &mockRecordRepo{records: make(map[string]*model.Record)}
}

func (r *mockRecordRepo) Save(_ context.Context, rec *model.Record) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[rec.TenantID+":"+rec.ID] = rec
	return nil
}

func (r *mockRecordRepo) FindByID(_ context.Context, tenantID, id string) (*model.Record, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.records[tenantID+":"+id]
	if !ok {
		return nil, fmt.Errorf("record not found")
	}
	return rec, nil
}

func (r *mockRecordRepo) FindBySearchToken(_ context.Context, _ string, _ string) ([]string, error) {
	return nil, nil
}

func (r *mockRecordRepo) ListByKeyVersion(_ context.Context, _ string, _ int, _, _ int) ([]*model.Record, error) {
	return nil, nil
}

func (r *mockRecordRepo) Update(_ context.Context, rec *model.Record) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[rec.TenantID+":"+rec.ID] = rec
	return nil
}

// mockAuditRepo discards all audit events (no side effects in unit tests).
type mockAuditRepo struct{}

func (a *mockAuditRepo) Append(_ context.Context, _ *model.AuditEvent) error { return nil }

// ── Helpers ─────────────────────────────────────────────────────────────────

func newTestService() *Service {
	hmacKey := make([]byte, 32)
	for i := range hmacKey {
		hmacKey[i] = 0xFF
	}
	cfg := &config.Config{HMACKey: hmacKey}
	log := logger.New("error") // suppress log output during tests
	return New(newMockHSM(), newMockCache(), newMockRecordRepo(), &mockAuditRepo{}, log, cfg)
}

// ── Tests ────────────────────────────────────────────────────────────────────

func TestEncryptDecryptRoundTrip(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	plaintext := []byte("sensitive certificate data")
	aad := []byte("tenant-a:record-test")

	encResp, err := svc.Encrypt(ctx, EncryptRequest{
		TenantID:  "tenant-a",
		Plaintext: plaintext,
		AAD:       aad,
		Actor:     "test",
		IPAddress: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if encResp.RecordID == "" {
		t.Fatal("RecordID must not be empty")
	}

	decResp, err := svc.Decrypt(ctx, DecryptRequest{
		TenantID:  "tenant-a",
		RecordID:  encResp.RecordID,
		Actor:     "test",
		IPAddress: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(decResp.Plaintext, plaintext) {
		t.Fatalf("plaintext mismatch\nwant: %q\ngot:  %q", plaintext, decResp.Plaintext)
	}
}

func TestEncryptDecryptWrongTenant(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	resp, err := svc.Encrypt(ctx, EncryptRequest{
		TenantID:  "tenant-a",
		Plaintext: []byte("secret"),
		AAD:       []byte("aad"),
		Actor:     "test",
	})
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Try to decrypt under a different tenant — should fail (record not found).
	_, err = svc.Decrypt(ctx, DecryptRequest{
		TenantID: "tenant-b",
		RecordID: resp.RecordID,
		Actor:    "test",
	})
	if err == nil {
		t.Fatal("expected error when decrypting under wrong tenant, got nil")
	}
}

func TestDecryptNonExistentRecord(t *testing.T) {
	svc := newTestService()
	_, err := svc.Decrypt(context.Background(), DecryptRequest{
		TenantID: "tenant-z",
		RecordID: "does-not-exist",
		Actor:    "test",
	})
	if err == nil {
		t.Fatal("expected error for missing record, got nil")
	}
}

func TestSearchTokenGeneratedDuringEncrypt(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	resp, err := svc.Encrypt(ctx, EncryptRequest{
		TenantID:     "tenant-s",
		Plaintext:    []byte("data"),
		AAD:          []byte("aad"),
		SearchFields: []string{"email@example.com", "ref-123"},
		Actor:        "test",
	})
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Retrieve the record and verify search tokens were saved.
	rec, err := svc.records.FindByID(ctx, "tenant-s", resp.RecordID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if len(rec.SearchTokens) != 2 {
		t.Fatalf("expected 2 search tokens, got %d", len(rec.SearchTokens))
	}
	for _, tok := range rec.SearchTokens {
		if len(tok) != 64 {
			t.Fatalf("search token should be 64-char hex string, got %q", tok)
		}
	}
}
