package hsm

import (
	"context"
	"crypto/aes"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/your-org/datavault/internal/domain/port"
)

// Stub is a software-only HSM implementation for development and testing.
// It stores KEKs in memory and uses AES key wrapping (RFC 3394).
//
// DEV/TEST ONLY — production use is explicitly blocked at runtime:
// hsm.New() returns a fatal error when DATAVAULT_HSM_MODE=stub and
// DATAVAULT_ENV=prod. There is no silent fallback.
// Replace with the PKCS#11 adapter for production (see pkcs11.go).
type Stub struct {
	mu   sync.RWMutex
	keks map[string]map[int][]byte // tenantID → keyVersion → KEK bytes
}

// NewStub creates a new in-memory HSM stub.
func NewStub() *Stub {
	return &Stub{
		keks: make(map[string]map[int][]byte),
	}
}

func (s *Stub) CurrentKeyVersion(_ context.Context, tenantID string) (int, error) {
	s.mu.RLock()
	if versions, ok := s.keks[tenantID]; ok && len(versions) > 0 {
		max := 0
		for v := range versions {
			if v > max {
				max = v
			}
		}
		s.mu.RUnlock()
		return max, nil
	}
	s.mu.RUnlock()

	// Tenant not seen yet — provision a new KEK under a write lock.
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check: another goroutine may have provisioned while we waited.
	if versions, ok := s.keks[tenantID]; ok && len(versions) > 0 {
		max := 0
		for v := range versions {
			if v > max {
				max = v
			}
		}
		return max, nil
	}

	kek := make([]byte, 32)
	if _, err := rand.Read(kek); err != nil {
		return 0, err
	}
	if s.keks[tenantID] == nil {
		s.keks[tenantID] = make(map[int][]byte)
	}
	s.keks[tenantID][1] = kek
	return 1, nil
}

func (s *Stub) WrapDEK(_ context.Context, tenantID string, keyVersion int, plaintextDEK []byte) ([]byte, error) {
	kek, err := s.getKEK(tenantID, keyVersion)
	if err != nil {
		return nil, err
	}
	return aesWrap(kek, plaintextDEK)
}

func (s *Stub) UnwrapDEK(_ context.Context, tenantID string, keyVersion int, wrappedDEK []byte) ([]byte, error) {
	kek, err := s.getKEK(tenantID, keyVersion)
	if err != nil {
		return nil, err
	}
	return aesUnwrap(kek, wrappedDEK)
}

// Ping implements port.HSMMonitor — the stub is always healthy (in-process).
func (s *Stub) Ping(_ context.Context) error { return nil }

// ── port.HSMMonitor — in-process stub implementations ────────────────────────

// NodeInfo returns zero-value counters. The stub does not perform real HSM ops.
func (s *Stub) NodeInfo(_ context.Context) (port.HSMNodeInfo, port.HSMSyncInfo, error) {
	s.mu.RLock()
	totalKeys := int64(0)
	for _, versions := range s.keks {
		totalKeys += int64(len(versions))
	}
	s.mu.RUnlock()
	return port.HSMNodeInfo{ID: 0, KeyCount: totalKeys}, port.HSMSyncInfo{}, nil
}

// ClusterInfo returns a single-node cluster entry representing this stub.
func (s *Stub) ClusterInfo(_ context.Context) ([]port.HSMClusterNode, error) {
	s.mu.RLock()
	totalKeys := int64(0)
	for _, versions := range s.keks {
		totalKeys += int64(len(versions))
	}
	s.mu.RUnlock()
	return []port.HSMClusterNode{{ID: 0, KeyCount: totalKeys}}, nil
}

// LogCount returns zeros — the stub has no persistent log storage.
func (s *Stub) LogCount(_ context.Context) (port.HSMLogCount, error) {
	return port.HSMLogCount{}, nil
}

// Date returns the current UTC time formatted as the CERTEX HSM ES date string.
func (s *Stub) Date(_ context.Context) (string, error) {
	return time.Now().UTC().Format("Mon Jan 02 15:04:05 +0000 2006"), nil
}

// Battery returns a healthy stub battery state with zero voltage.
func (s *Stub) Battery(_ context.Context) (port.HSMBattery, error) {
	return port.HSMBattery{NeedReplace: false, VoltageMillivolts: 0}, nil
}

// NTPStatus always returns the stub indicator — NTP is not applicable in-process.
func (s *Stub) NTPStatus(_ context.Context) (string, error) {
	return "stub - not applicable", nil
}

// ActiveKeys returns an empty slice — the stub processes all operations synchronously.
func (s *Stub) ActiveKeys(_ context.Context) ([]string, error) {
	return []string{}, nil
}

func (s *Stub) getKEK(tenantID string, keyVersion int) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if versions, ok := s.keks[tenantID]; ok {
		if kek, ok := versions[keyVersion]; ok {
			return kek, nil
		}
	}
	return nil, fmt.Errorf("hsm stub: KEK not found for tenant %s version %d", tenantID, keyVersion)
}

// aesWrap implements RFC 3394 AES Key Wrap.
// kek must be 16, 24, or 32 bytes; dek must be a multiple of 8 bytes.
// Output is len(dek)+8 bytes (includes 8-byte integrity check value).
func aesWrap(kek, dek []byte) ([]byte, error) {
	if len(dek)%8 != 0 || len(dek) < 16 {
		return nil, fmt.Errorf("hsm: dek length must be a multiple of 8 and at least 16 bytes")
	}
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}
	n := len(dek) / 8

	// iv is the standard RFC 3394 integrity check value.
	const iv = uint64(0xA6A6A6A6A6A6A6A6)

	// R[0..n-1] holds the 8-byte key data blocks.
	R := make([]uint64, n)
	for i := range R {
		R[i] = beUint64(dek[i*8:])
	}

	A := iv
	buf := make([]byte, 16)
	for j := 0; j <= 5; j++ {
		for i := 0; i < n; i++ {
			putBeUint64(buf[0:], A)
			putBeUint64(buf[8:], R[i])
			block.Encrypt(buf, buf)
			t := uint64(n*j + i + 1)
			A = beUint64(buf[0:]) ^ t
			R[i] = beUint64(buf[8:])
		}
	}

	out := make([]byte, 8+len(dek))
	putBeUint64(out, A)
	for i, v := range R {
		putBeUint64(out[8+i*8:], v)
	}
	return out, nil
}

// aesUnwrap implements RFC 3394 AES Key Unwrap.
// wrapped must be len(dek)+8 bytes (as produced by aesWrap).
func aesUnwrap(kek, wrapped []byte) ([]byte, error) {
	if len(wrapped)%8 != 0 || len(wrapped) < 24 {
		return nil, fmt.Errorf("hsm: wrapped key length must be a multiple of 8 and at least 24 bytes")
	}
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}
	n := (len(wrapped) / 8) - 1

	const iv = uint64(0xA6A6A6A6A6A6A6A6)

	A := beUint64(wrapped)
	R := make([]uint64, n)
	for i := range R {
		R[i] = beUint64(wrapped[8+i*8:])
	}

	buf := make([]byte, 16)
	for j := 5; j >= 0; j-- {
		for i := n - 1; i >= 0; i-- {
			t := uint64(n*j + i + 1)
			putBeUint64(buf[0:], A^t)
			putBeUint64(buf[8:], R[i])
			block.Decrypt(buf, buf)
			A = beUint64(buf[0:])
			R[i] = beUint64(buf[8:])
		}
	}

	if A != iv {
		return nil, fmt.Errorf("hsm: key unwrap failed — integrity check mismatch")
	}

	out := make([]byte, n*8)
	for i, v := range R {
		putBeUint64(out[i*8:], v)
	}
	return out, nil
}

// beUint64 reads a big-endian uint64 from b.
func beUint64(b []byte) uint64 {
	return uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
		uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
}

// putBeUint64 writes v as big-endian uint64 into b.
func putBeUint64(b []byte, v uint64) {
	b[0] = byte(v >> 56)
	b[1] = byte(v >> 48)
	b[2] = byte(v >> 40)
	b[3] = byte(v >> 32)
	b[4] = byte(v >> 24)
	b[5] = byte(v >> 16)
	b[6] = byte(v >> 8)
	b[7] = byte(v)
}
