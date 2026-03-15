package port

import "context"

// HSMMonitor is the outbound port for operational HSM monitoring.
// It provides health checks and diagnostic telemetry from the HSM node.
// All adapters returned by hsm.New() implement this interface.
//
// Source: CERTEX HSM ES Программный интерфейс (API), 398-980340000290.HSM 90.01.22120746
type HSMMonitor interface {
	// Ping checks HSM connectivity and Basic-auth authorization.
	Ping(ctx context.Context) error

	// NodeInfo returns the current node operation counters and sync state
	// (POST /info).
	NodeInfo(ctx context.Context) (HSMNodeInfo, HSMSyncInfo, error)

	// ClusterInfo returns key counts from every cluster node
	// (POST /infocluster). Unreachable nodes have KeyCount == -1.
	ClusterInfo(ctx context.Context) ([]HSMClusterNode, error)

	// LogCount returns extended state-log record statistics from the node
	// (POST /logcount).
	LogCount(ctx context.Context) (HSMLogCount, error)

	// Date returns the date/time string as reported by the HSM node
	// (POST /date).
	Date(ctx context.Context) (string, error)

	// Battery returns the battery state of the HSM node (POST /battery).
	Battery(ctx context.Context) (HSMBattery, error)

	// NTPStatus triggers an NTP sync check and returns the raw HSM status
	// string (POST /updatetime).
	// "200 OK" — NTP sync is configured.
	// "506 Cannot talk to daemon" — NTP sync is not configured.
	NTPStatus(ctx context.Context) (string, error)

	// ActiveKeys returns the names of keys currently being processed on
	// the node (POST /infogen). Returns an empty slice when idle.
	ActiveKeys(ctx context.Context) ([]string, error)
}

// HSMNodeInfo holds per-node telemetry counters from POST /info (info section).
type HSMNodeInfo struct {
	ID             int   `json:"id"`
	KeyCount       int64 `json:"key_count"`
	FKeyDelTotal   int64 `json:"fkey_del_total"`
	FKeyDelCount   int64 `json:"fkey_del_count"`
	FKeyDelError   int64 `json:"fkey_del_error"`
	FKeyDelTimeUs  int64 `json:"fkey_del_time_us"`
	FKeyGenTotal   int64 `json:"fkey_gen_total"`
	FKeyGenCount   int64 `json:"fkey_gen_count"`
	FKeyGenError   int64 `json:"fkey_gen_error"`
	FKeyGenTimeUs  int64 `json:"fkey_gen_time_us"`
	FKeySignTotal  int64 `json:"fkey_sign_total"`
	FKeySignCount  int64 `json:"fkey_sign_count"`
	FKeySignError  int64 `json:"fkey_sign_error"`
	FKeySignTimeUs int64 `json:"fkey_sign_time_us"`
	TasksQueue     int   `json:"tasks_queue"`
	TasksQueueNet  int   `json:"tasks_queue_net"`
}

// HSMSyncInfo holds inter-node synchronisation telemetry from POST /info
// (sync section).
type HSMSyncInfo struct {
	CountSyncKey       int   `json:"count_sync_key"`
	CountSyncState     int   `json:"count_sync_state"`
	PercentSyncKey     int   `json:"percent_sync_key"`
	PercentSyncState   int   `json:"percent_sync_state"`
	SyncProcess        bool  `json:"sync_process"`
	SyncTimeMs         int64 `json:"sync_time_ms"`
	SyncKeyID          int   `json:"sync_key_id"`
	SyncStateID        int   `json:"sync_state_id"`
	TaskSendCnt        int64 `json:"task_send_cnt"`
	TaskUpdateCnt      int64 `json:"task_update_cnt"`
	TaskUpdateCntStart int64 `json:"task_update_cnt_start"`
}

// HSMClusterNode holds per-node key count from POST /infocluster.
type HSMClusterNode struct {
	ID       int   `json:"id"`
	KeyCount int64 `json:"key_count"`
}

// HSMBattery holds battery state from POST /battery.
type HSMBattery struct {
	NeedReplace       bool `json:"need_replace"`
	VoltageMillivolts int  `json:"voltage_millivolts"`
}

// HSMLogCount holds extended log-record statistics from POST /logcount.
type HSMLogCount struct {
	DBTotal  int64 `json:"db_total"`
	Deleted  int64 `json:"deleted"`
	Active   int64 `json:"active"`
	InMemory int64 `json:"in_memory"`
}
