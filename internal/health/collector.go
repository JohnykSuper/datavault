// Package health collects service, runtime, and infrastructure status
// for the unified GET /health endpoint.
package health

import (
	"context"
	"os"
	"runtime"
	"time"

	"github.com/your-org/datavault/internal/config"
	"github.com/your-org/datavault/internal/domain/port"
	"github.com/your-org/datavault/internal/version"
)

// CacheStatser is satisfied by *cache.DEKCache.
type CacheStatser interface {
	ItemCount() int
	CacheTTL() time.Duration
}

// ── Response types ────────────────────────────────────────────────────────────

// ServiceInfo describes the static service configuration.
type ServiceInfo struct {
	Env      string `json:"env"`
	HSMMode  string `json:"hsm_mode"`
	DBDriver string `json:"db_driver"`
	Hostname string `json:"hostname"`
}

// RuntimeInfo holds live Go-runtime metrics.
type RuntimeInfo struct {
	GoVersion  string `json:"go_version"`
	Goroutines int    `json:"goroutines"`
	CPUs       int    `json:"cpus"`
}

// MemoryInfo holds Go heap and GC statistics (values in megabytes).
type MemoryInfo struct {
	AllocMB      float64 `json:"alloc_mb"`
	TotalAllocMB float64 `json:"total_alloc_mb"`
	SysMB        float64 `json:"sys_mb"`
	HeapAllocMB  float64 `json:"heap_alloc_mb"`
	HeapSysMB    float64 `json:"heap_sys_mb"`
	GCCycles     uint32  `json:"gc_cycles"`
}

// DBStatus is the health status of the database.
type DBStatus struct {
	Status    string `json:"status"`
	Driver    string `json:"driver"`
	Detail    string `json:"detail,omitempty"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
}

// HSMStatus is the complete health and telemetry snapshot of the HSM.
type HSMStatus struct {
	Status     string                `json:"status"`
	Mode       string                `json:"mode"`
	Detail     string                `json:"detail,omitempty"`
	LatencyMs  int64                 `json:"latency_ms,omitempty"`
	Node       *port.HSMNodeInfo     `json:"node,omitempty"`
	Sync       *port.HSMSyncInfo     `json:"sync,omitempty"`
	Cluster    []port.HSMClusterNode `json:"cluster,omitempty"`
	LogCount   *port.HSMLogCount     `json:"log_count,omitempty"`
	Battery    *port.HSMBattery      `json:"battery,omitempty"`
	NodeTime   string                `json:"node_time,omitempty"`
	NTPStatus  string                `json:"ntp_status,omitempty"`
	ActiveKeys []string              `json:"active_keys"`
	Errors     []string              `json:"errors,omitempty"`
}

// CacheStatus is the health status of the in-process DEK cache.
type CacheStatus struct {
	Status string `json:"status"`
	Items  int    `json:"items"`
	TTL    string `json:"ttl"`
}

// Components groups all infrastructure component statuses.
type Components struct {
	DB       DBStatus    `json:"db"`
	HSM      HSMStatus   `json:"hsm"`
	DEKCache CacheStatus `json:"dek_cache"`
}

// HealthResponse is the unified wire format for GET /health.
// Status is "ok" only when every component reports healthy.
type HealthResponse struct {
	Status     string      `json:"status"`
	Version    string      `json:"version"`
	Time       string      `json:"time"`
	Uptime     string      `json:"uptime"`
	Service    ServiceInfo `json:"service"`
	Runtime    RuntimeInfo `json:"runtime"`
	Memory     MemoryInfo  `json:"memory"`
	Components Components  `json:"components"`
}

// ── Collector ─────────────────────────────────────────────────────────────────

// Collector assembles the comprehensive health snapshot.
type Collector struct {
	startTime time.Time
	dbPinger  port.Pinger
	hsm       port.HSMMonitor
	cache     CacheStatser
	cfg       *config.Config
}

// New creates a Collector. It records time.Now() as the service start time.
func New(cfg *config.Config, dbPinger port.Pinger, hsm port.HSMMonitor, cache CacheStatser) *Collector {
	return &Collector{
		startTime: time.Now(),
		dbPinger:  dbPinger,
		hsm:       hsm,
		cache:     cache,
		cfg:       cfg,
	}
}

// Check performs all health checks and returns a snapshot.
// The second return value is true when every component is healthy.
// A 5-second context deadline is strongly recommended by the caller.
func (c *Collector) Check(ctx context.Context) (HealthResponse, bool) {
	db := c.checkDB(ctx)
	hsm := c.checkHSM(ctx)
	cache := c.checkCache()

	allOK := db.Status == "ok" && hsm.Status == "ok"
	status := "ok"
	if !allOK {
		status = "error"
	}

	return HealthResponse{
		Status:  status,
		Version: version.Version,
		Time:    nowRFC3339(),
		Uptime:  uptime(c.startTime),
		Service: c.serviceInfo(),
		Runtime: collectRuntime(),
		Memory:  collectMemory(),
		Components: Components{
			DB:       db,
			HSM:      hsm,
			DEKCache: cache,
		},
	}, allOK
}

// ── component checks ──────────────────────────────────────────────────────────

func (c *Collector) checkDB(ctx context.Context) DBStatus {
	t0 := time.Now()
	if err := c.dbPinger.Ping(ctx); err != nil {
		return DBStatus{Status: "error", Driver: c.cfg.DBDriver, Detail: err.Error()}
	}
	return DBStatus{
		Status:    "ok",
		Driver:    c.cfg.DBDriver,
		LatencyMs: time.Since(t0).Milliseconds(),
	}
}

func (c *Collector) checkHSM(ctx context.Context) HSMStatus {
	// ── 1. Connectivity check ──────────────────────────────────────────────
	t0 := time.Now()
	if err := c.hsm.Ping(ctx); err != nil {
		return HSMStatus{
			Status:     "error",
			Mode:       c.cfg.HSMMode,
			Detail:     err.Error(),
			ActiveKeys: []string{},
		}
	}
	latency := time.Since(t0).Milliseconds()

	// ── 2. Collect detailed telemetry (failures are non-fatal) ────────────
	var errs []string

	node, sync, err := c.hsm.NodeInfo(ctx)
	var nodePtr *port.HSMNodeInfo
	var syncPtr *port.HSMSyncInfo
	if err != nil {
		errs = append(errs, "node_info: "+err.Error())
	} else {
		nodePtr = &node
		syncPtr = &sync
	}

	cluster, err := c.hsm.ClusterInfo(ctx)
	if err != nil {
		errs = append(errs, "cluster_info: "+err.Error())
	}

	logCount, err := c.hsm.LogCount(ctx)
	var logCountPtr *port.HSMLogCount
	if err != nil {
		errs = append(errs, "log_count: "+err.Error())
	} else {
		logCountPtr = &logCount
	}

	battery, err := c.hsm.Battery(ctx)
	var batteryPtr *port.HSMBattery
	if err != nil {
		errs = append(errs, "battery: "+err.Error())
	} else {
		batteryPtr = &battery
	}

	nodeTime, err := c.hsm.Date(ctx)
	if err != nil {
		errs = append(errs, "date: "+err.Error())
	}

	ntpStatus, err := c.hsm.NTPStatus(ctx)
	if err != nil {
		errs = append(errs, "ntp_status: "+err.Error())
	}

	activeKeys, err := c.hsm.ActiveKeys(ctx)
	if err != nil {
		errs = append(errs, "active_keys: "+err.Error())
		activeKeys = []string{}
	}

	return HSMStatus{
		Status:     "ok",
		Mode:       c.cfg.HSMMode,
		LatencyMs:  latency,
		Node:       nodePtr,
		Sync:       syncPtr,
		Cluster:    cluster,
		LogCount:   logCountPtr,
		Battery:    batteryPtr,
		NodeTime:   nodeTime,
		NTPStatus:  ntpStatus,
		ActiveKeys: activeKeys,
		Errors:     errs,
	}
}

func (c *Collector) checkCache() CacheStatus {
	return CacheStatus{
		Status: "ok",
		Items:  c.cache.ItemCount(),
		TTL:    c.cache.CacheTTL().String(),
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (c *Collector) serviceInfo() ServiceInfo {
	hostname, _ := os.Hostname()
	return ServiceInfo{
		Env:      c.cfg.Env,
		HSMMode:  c.cfg.HSMMode,
		DBDriver: c.cfg.DBDriver,
		Hostname: hostname,
	}
}

func collectRuntime() RuntimeInfo {
	return RuntimeInfo{
		GoVersion:  runtime.Version(),
		Goroutines: runtime.NumGoroutine(),
		CPUs:       runtime.NumCPU(),
	}
}

func collectMemory() MemoryInfo {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return MemoryInfo{
		AllocMB:      toMB(ms.Alloc),
		TotalAllocMB: toMB(ms.TotalAlloc),
		SysMB:        toMB(ms.Sys),
		HeapAllocMB:  toMB(ms.HeapAlloc),
		HeapSysMB:    toMB(ms.HeapSys),
		GCCycles:     ms.NumGC,
	}
}

func nowRFC3339() string { return time.Now().UTC().Format(time.RFC3339) }

func uptime(start time.Time) string { return time.Since(start).Truncate(time.Second).String() }

func toMB(b uint64) float64 { return float64(b) / 1024 / 1024 }
