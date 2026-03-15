// Package hsm — CertexREST adapter for CERTEX HSM ES via the verified REST API.
//
// Verified API surface (source: 398-980340000290.HSM 90.01.22120746):
//
//	POST /info        — node telemetry + sync state
//	POST /infocluster — cluster-wide key counts
//	POST /infogen     — keys currently being processed
//	POST /infolog     — log record counts
//	POST /logcount    — extended log statistics
//	POST /findkey/{name} — key lookup by name
//	POST /date        — node date/time
//	POST /battery     — battery state
//	POST /updatetime  — NTP sync check
//	POST /clear       — clear statistical counters
//
// DEK wrap/unwrap is NOT implemented — the vendor API document does not
// describe cryptographic key-operation endpoints. These will be added once
// verified vendor documentation is available.
package hsm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/your-org/datavault/internal/domain/port"
)

// CertexREST is the CERTEX HSM ES REST adapter.
// It implements port.HSMMonitor (all monitoring/diagnostic endpoints).
// port.HSM crypto methods (WrapDEK, UnwrapDEK, CurrentKeyVersion) return
// ErrNotImplemented until vendor crypto documentation is available.
type CertexREST struct {
	baseURL  string // e.g. https://10.0.0.1:8443
	username string
	password string
	client   *http.Client
}

// NewCertexREST creates a CertexREST adapter.
// baseURL must be the HTTPS base URL of the CERTEX HSM ES node.
// username and password are used for HTTP Basic authorization.
func NewCertexREST(baseURL, username, password string) *CertexREST {
	return &CertexREST{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		password: password,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ── port.HSM — crypto operations (not yet implemented) ───────────────────────

// WrapDEK is not implemented — pending verified vendor crypto documentation.
func (c *CertexREST) WrapDEK(_ context.Context, _ string, _ int, _ []byte) ([]byte, error) {
	return nil, fmt.Errorf("certex: WrapDEK is not implemented — " +
		"waiting for verified vendor API documentation for key wrap operations")
}

// UnwrapDEK is not implemented — pending verified vendor crypto documentation.
func (c *CertexREST) UnwrapDEK(_ context.Context, _ string, _ int, _ []byte) ([]byte, error) {
	return nil, fmt.Errorf("certex: UnwrapDEK is not implemented — " +
		"waiting for verified vendor API documentation for key unwrap operations")
}

// CurrentKeyVersion is not implemented — pending verified vendor documentation.
func (c *CertexREST) CurrentKeyVersion(_ context.Context, _ string) (int, error) {
	return 0, fmt.Errorf("certex: CurrentKeyVersion is not implemented — " +
		"waiting for verified vendor API documentation for key management operations")
}

// ── port.HSMMonitor — monitoring operations ───────────────────────────────────

// Ping performs POST /info and verifies that the response is valid JSON,
// confirming HSM connectivity and Basic-Auth credentials.
func (c *CertexREST) Ping(ctx context.Context) error {
	_, _, err := c.NodeInfo(ctx)
	return err
}

// NodeInfo calls POST /info and returns node telemetry and sync state.
func (c *CertexREST) NodeInfo(ctx context.Context) (port.HSMNodeInfo, port.HSMSyncInfo, error) {
	var resp certexInfoResp
	if err := c.post(ctx, "/info", &resp); err != nil {
		return port.HSMNodeInfo{}, port.HSMSyncInfo{}, fmt.Errorf("certex /info: %w", err)
	}
	node := port.HSMNodeInfo{
		ID:             resp.Info.ID,
		KeyCount:       resp.Info.Key,
		FKeyDelTotal:   resp.Info.FKeyDelTotal,
		FKeyDelCount:   resp.Info.FKeyDelCount,
		FKeyDelError:   resp.Info.FKeyDelError,
		FKeyDelTimeUs:  resp.Info.FKeyDelTime,
		FKeyGenTotal:   resp.Info.FKeyGenTotal,
		FKeyGenCount:   resp.Info.FKeyGenCount,
		FKeyGenError:   resp.Info.FKeyGenError,
		FKeyGenTimeUs:  resp.Info.FKeyGenTime,
		FKeySignTotal:  resp.Info.FKeySignTotal,
		FKeySignCount:  resp.Info.FKeySignCount,
		FKeySignError:  resp.Info.FKeySignError,
		FKeySignTimeUs: resp.Info.FKeySignTime,
		TasksQueue:     resp.Info.TasksQueue,
		TasksQueueNet:  resp.Info.TasksQueueNet,
	}
	sync := port.HSMSyncInfo{
		CountSyncKey:       resp.Sync.CountSyncKey,
		CountSyncState:     resp.Sync.CountSyncState,
		PercentSyncKey:     resp.Sync.PercentSyncKey,
		PercentSyncState:   resp.Sync.PercentSyncState,
		SyncProcess:        resp.Sync.SyncProcess,
		SyncTimeMs:         resp.Sync.SyncTime,
		SyncKeyID:          resp.Sync.SyncKeyID,
		SyncStateID:        resp.Sync.SyncStateID,
		TaskSendCnt:        resp.Sync.TaskSendCnt,
		TaskUpdateCnt:      resp.Sync.TaskUpdateCnt,
		TaskUpdateCntStart: resp.Sync.TaskUpdateCntStart,
	}
	return node, sync, nil
}

// ClusterInfo calls POST /infocluster and returns key counts per cluster node.
func (c *CertexREST) ClusterInfo(ctx context.Context) ([]port.HSMClusterNode, error) {
	var resp certexClusterResp
	if err := c.post(ctx, "/infocluster", &resp); err != nil {
		return nil, fmt.Errorf("certex /infocluster: %w", err)
	}
	nodes := make([]port.HSMClusterNode, len(resp.Info))
	for i, n := range resp.Info {
		nodes[i] = port.HSMClusterNode{ID: n.ID, KeyCount: n.Key}
	}
	return nodes, nil
}

// LogCount calls POST /logcount and returns extended log statistics.
func (c *CertexREST) LogCount(ctx context.Context) (port.HSMLogCount, error) {
	var resp certexLogCountResp
	if err := c.post(ctx, "/logcount", &resp); err != nil {
		return port.HSMLogCount{}, fmt.Errorf("certex /logcount: %w", err)
	}
	return port.HSMLogCount{
		DBTotal:  resp.DB,
		Deleted:  resp.Del,
		Active:   resp.Gen,
		InMemory: resp.Log,
	}, nil
}

// Date calls POST /date and returns the node's current date/time string.
func (c *CertexREST) Date(ctx context.Context) (string, error) {
	var resp certexDateResp
	if err := c.post(ctx, "/date", &resp); err != nil {
		return "", fmt.Errorf("certex /date: %w", err)
	}
	return resp.Date, nil
}

// Battery calls POST /battery and returns the battery state.
func (c *CertexREST) Battery(ctx context.Context) (port.HSMBattery, error) {
	var resp certexBatteryResp
	if err := c.post(ctx, "/battery", &resp); err != nil {
		return port.HSMBattery{}, fmt.Errorf("certex /battery: %w", err)
	}
	return port.HSMBattery{
		NeedReplace:       resp.NeedReplace,
		VoltageMillivolts: resp.Voltage,
	}, nil
}

// NTPStatus calls POST /updatetime and returns the raw NTP status string.
func (c *CertexREST) NTPStatus(ctx context.Context) (string, error) {
	var resp certexReturnResp
	if err := c.post(ctx, "/updatetime", &resp); err != nil {
		return "", fmt.Errorf("certex /updatetime: %w", err)
	}
	return resp.Return, nil
}

// ActiveKeys calls POST /infogen and returns names of keys being processed.
func (c *CertexREST) ActiveKeys(ctx context.Context) ([]string, error) {
	var resp certexInfoGenResp
	if err := c.post(ctx, "/infogen", &resp); err != nil {
		return nil, fmt.Errorf("certex /infogen: %w", err)
	}
	if resp.Name == nil {
		return []string{}, nil
	}
	return resp.Name, nil
}

// ── internal HTTP helper ──────────────────────────────────────────────────────

// post sends an authenticated POST to the given path and decodes the JSON body.
// The Authorization header is never logged.
func (c *CertexREST) post(ctx context.Context, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, http.NoBody)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.username, c.password) // never log credentials
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return json.Unmarshal(body, out)
}

// ── vendor JSON DTOs ──────────────────────────────────────────────────────────
// These types mirror the exact JSON field names from the vendor API document.
// They are kept inside the hsm package and never leak into domain logic.

type certexInfoResp struct {
	Info struct {
		FKeyDelCount  int64 `json:"FKeyDelCount"`
		FKeyDelError  int64 `json:"FKeyDelError"`
		FKeyDelTime   int64 `json:"FKeyDelTime"`
		FKeyDelTotal  int64 `json:"FKeyDelTotal"`
		FKeyGenCount  int64 `json:"FKeyGenCount"`
		FKeyGenError  int64 `json:"FKeyGenError"`
		FKeyGenTime   int64 `json:"FKeyGenTime"`
		FKeyGenTotal  int64 `json:"FKeyGenTotal"`
		FKeySignCount int64 `json:"FKeySignCount"`
		FKeySignError int64 `json:"FKeySignError"`
		FKeySignTime  int64 `json:"FKeySignTime"`
		FKeySignTotal int64 `json:"FKeySignTotal"`
		ID            int   `json:"id"`
		Key           int64 `json:"key"`
		TasksQueue    int   `json:"tasksQueue"`
		TasksQueueNet int   `json:"tasksQueueNet"`
	} `json:"info"`
	Sync struct {
		CountSyncKey       int   `json:"count_sync_key"`
		CountSyncState     int   `json:"count_sync_state"`
		PercentSyncKey     int   `json:"percent_sync_key"`
		PercentSyncState   int   `json:"percent_sync_state"`
		SyncProcess        bool  `json:"syncProcess"`
		SyncTime           int64 `json:"syncTime"`
		SyncKeyID          int   `json:"sync_key_id"`
		SyncStateID        int   `json:"sync_state_id"`
		TaskSendCnt        int64 `json:"task_send_cnt"`
		TaskUpdateCnt      int64 `json:"task_update_cnt"`
		TaskUpdateCntStart int64 `json:"task_update_cnt_start"`
	} `json:"sync"`
}

type certexClusterResp struct {
	Info []struct {
		ID  int   `json:"id"`
		Key int64 `json:"key"`
	} `json:"info"`
}

type certexLogCountResp struct {
	DB  int64 `json:"db"`
	Del int64 `json:"del"`
	Gen int64 `json:"gen"`
	Log int64 `json:"log"`
}

type certexDateResp struct {
	Date string `json:"date"`
}

type certexBatteryResp struct {
	NeedReplace bool `json:"need_replace"`
	Voltage     int  `json:"voltage"`
}

type certexReturnResp struct {
	Return string `json:"return"`
}

type certexInfoGenResp struct {
	Name []string `json:"name"`
}
