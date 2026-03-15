// Package health collects service, runtime, and infrastructure status
// for the /health and /ready HTTP probes.
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

// ── Response types ───────────────────────────────────────────────────────────

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

// MemoryInfo holds Go memory statistics.
type MemoryInfo struct {
	AllocMB      float64 `json:"alloc_mb"`
	TotalAllocMB float64 `json:"total_alloc_mb"`
	SysMB        float64 `json:"sys_mb"`
	HeapAllocMB  float64 `json:"heap_alloc_mb"`
	HeapSysMB    float64 `json:"heap_sys_mb"`
	GCCycles     uint32  `json:"gc_cycles"`
}

// ComponentStatus is the health status of an external dependency.
type ComponentStatus struct {
	Status    string `json:"status"`
	Detail    string `json:"detail,omitempty"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
}

// CacheInfo is the health status of the in-process DEK cache.
type CacheInfo struct {
	Status string `json:"status"`
	Items  int    `json:"items"`
	TTL    string `json:"ttl"`
}

// ReadyComponents groups all component statuses.
type ReadyComponents struct {
	DB       ComponentStatus `json:"db"`
	HSM      ComponentStatus `json:"hsm"`
	DEKCache CacheInfo       `json:"dek_cache"`
}

// HealthResponse is the wire format for GET /health (liveness only).
type HealthResponse struct {
	Status  string      `json:"status"`
	Version string      `json:"version"`
	Time    string      `json:"time"`
	Uptime  string      `json:"uptime"`
	Service ServiceInfo `json:"service"`
	Runtime RuntimeInfo `json:"runtime"`
	Memory  MemoryInfo  `json:"memory"`
}

// ReadyResponse is the wire format for GET /ready (readiness, incl. probes).
type ReadyResponse struct {
	Status     string          `json:"status"`
	Version    string          `json:"version"`
	Time       string          `json:"time"`
	Uptime     string          `json:"uptime"`
	Service    ServiceInfo     `json:"service"`
	Runtime    RuntimeInfo     `json:"runtime"`
	Memory     MemoryInfo      `json:"memory"`
	Components ReadyComponents `json:"components"`
}

// ── Collector ────────────────────────────────────────────────────────────────

// Collector assembles health and readiness snapshots.
type Collector struct {
	startTime time.Time
	dbPinger  port.Pinger
	hsmPinger port.Pinger
	cache     CacheStatser
	cfg       *config.Config
}

// New creates a Collector. It records the current time as the service start time.
func New(cfg *config.Config, dbPinger port.Pinger, hsmPinger port.Pinger, cache CacheStatser) *Collector {
	return &Collector{
		startTime: time.Now(),
		dbPinger:  dbPinger,
		hsmPinger: hsmPinger,
		cache:     cache,
		cfg:       cfg,
	}
}

// Health returns a liveness snapshot without making any external calls.
func (c *Collector) Health() HealthResponse {
	return HealthResponse{
		Status:  "ok",
		Version: version.Version,
		Time:    nowRFC3339(),
		Uptime:  uptime(c.startTime),
		Service: c.serviceInfo(),
		Runtime: collectRuntime(),
		Memory:  collectMemory(),
	}
}

// Ready runs DB and HSM ping checks and returns a readiness snapshot.
// The second return value is true when every component reports "ok".
func (c *Collector) Ready(ctx context.Context) (ReadyResponse, bool) {
	dbStatus := c.probe(ctx, c.dbPinger)
	hsmStatus := c.probe(ctx, c.hsmPinger)

	cacheInfo := CacheInfo{
		Status: "ok",
		Items:  c.cache.ItemCount(),
		TTL:    c.cache.CacheTTL().String(),
	}

	allOK := dbStatus.Status == "ok" && hsmStatus.Status == "ok"
	status := "ready"
	if !allOK {
		status = "unavailable"
	}

	return ReadyResponse{
		Status:  status,
		Version: version.Version,
		Time:    nowRFC3339(),
		Uptime:  uptime(c.startTime),
		Service: c.serviceInfo(),
		Runtime: collectRuntime(),
		Memory:  collectMemory(),
		Components: ReadyComponents{
			DB:       dbStatus,
			HSM:      hsmStatus,
			DEKCache: cacheInfo,
		},
	}, allOK
}

// ── internal helpers ─────────────────────────────────────────────────────────

func (c *Collector) probe(ctx context.Context, p port.Pinger) ComponentStatus {
	t0 := time.Now()
	if err := p.Ping(ctx); err != nil {
		return ComponentStatus{Status: "error", Detail: err.Error()}
	}
	return ComponentStatus{Status: "ok", LatencyMs: time.Since(t0).Milliseconds()}
}

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

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func uptime(start time.Time) string {
	return time.Since(start).Truncate(time.Second).String()
}

func toMB(b uint64) float64 {
	return float64(b) / 1024 / 1024
}
