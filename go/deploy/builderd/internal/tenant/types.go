package tenant

import (
	"time"
)

// QuotaType represents different types of quotas
type QuotaType string

const (
	QuotaTypeConcurrentBuilds QuotaType = "concurrent_builds"
	QuotaTypeDailyBuilds      QuotaType = "daily_builds"
	QuotaTypeStorage          QuotaType = "storage"
	QuotaTypeCompute          QuotaType = "compute"
	QuotaTypeBuildTime        QuotaType = "build_time"
	QuotaTypeMemory           QuotaType = "memory"
	QuotaTypeCPU              QuotaType = "cpu"
	QuotaTypeDisk             QuotaType = "disk"
	QuotaTypeNetwork          QuotaType = "network"
)

// QuotaError represents a quota violation error
type QuotaError struct {
	Type     QuotaType `json:"type"`
	TenantID string    `json:"tenant_id"`
	Current  int64     `json:"current"`
	Limit    int64     `json:"limit"`
	Message  string    `json:"message"`
}

// Error implements the error interface
func (e *QuotaError) Error() string {
	return e.Message
}

// IsQuotaError checks if an error is a quota error
func IsQuotaError(err error) bool {
	_, ok := err.(*QuotaError)
	return ok
}

// UsageStats represents current usage statistics for a tenant
type UsageStats struct {
	TenantID           string    `json:"tenant_id"`
	ActiveBuilds       int32     `json:"active_builds"`
	DailyBuildsUsed    int32     `json:"daily_builds_used"`
	StorageBytesUsed   int64     `json:"storage_bytes_used"`
	ComputeMinutesUsed int64     `json:"compute_minutes_used"`
	Timestamp          time.Time `json:"timestamp"`
}

// BuildConstraints represents resource constraints for a specific build
type BuildConstraints struct {
	// Process constraints
	MaxMemoryBytes int64 `json:"max_memory_bytes"`
	MaxCPUCores    int32 `json:"max_cpu_cores"`
	MaxDiskBytes   int64 `json:"max_disk_bytes"`
	TimeoutSeconds int32 `json:"timeout_seconds"`

	// Security constraints
	RunAsUser           int32    `json:"run_as_user"`
	RunAsGroup          int32    `json:"run_as_group"`
	ReadOnlyRootfs      bool     `json:"read_only_rootfs"`
	NoPrivileged        bool     `json:"no_privileged"`
	DroppedCapabilities []string `json:"dropped_capabilities"`

	// Network constraints
	NetworkMode       string   `json:"network_mode"`
	AllowedRegistries []string `json:"allowed_registries"`
	AllowedGitHosts   []string `json:"allowed_git_hosts"`
	BlockedDomains    []string `json:"blocked_domains"`

	// Storage constraints
	WorkspaceDir     string `json:"workspace_dir"`
	RootfsDir        string `json:"rootfs_dir"`
	TempDir          string `json:"temp_dir"`
	MaxTempSizeBytes int64  `json:"max_temp_size_bytes"`
}

// IsolationLevel represents the level of isolation for a tenant
type IsolationLevel int

const (
	IsolationLevelNone IsolationLevel = iota
	IsolationLevelBasic
	IsolationLevelStrict
	IsolationLevelMaximum
)

// String returns the string representation of an isolation level
func (l IsolationLevel) String() string {
	switch l {
	case IsolationLevelNone:
		return "none"
	case IsolationLevelBasic:
		return "basic"
	case IsolationLevelStrict:
		return "strict"
	case IsolationLevelMaximum:
		return "maximum"
	default:
		return "unknown"
	}
}

// SecurityPolicy represents security policies for a tenant
type SecurityPolicy struct {
	IsolationLevel      IsolationLevel `json:"isolation_level"`
	AllowPrivileged     bool           `json:"allow_privileged"`
	AllowHostNetwork    bool           `json:"allow_host_network"`
	AllowHostPID        bool           `json:"allow_host_pid"`
	AllowHostIPC        bool           `json:"allow_host_ipc"`
	AllowSysAdmin       bool           `json:"allow_sys_admin"`
	RequireNonRoot      bool           `json:"require_non_root"`
	SelinuxEnabled      bool           `json:"selinux_enabled"`
	AppArmorEnabled     bool           `json:"apparmor_enabled"`
	SeccompProfile      string         `json:"seccomp_profile"`
	DroppedCapabilities []string       `json:"dropped_capabilities"`
	AddedCapabilities   []string       `json:"added_capabilities"`
}

// AuditEvent represents an audit event for compliance tracking
type AuditEvent struct {
	EventID    string                 `json:"event_id"`
	TenantID   string                 `json:"tenant_id"`
	CustomerID string                 `json:"customer_id"`
	BuildID    string                 `json:"build_id,omitempty"`
	Action     string                 `json:"action"`
	Resource   string                 `json:"resource"`
	Result     string                 `json:"result"`
	Reason     string                 `json:"reason,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// AuditAction represents different types of audit actions
type AuditAction string

const (
	AuditActionBuildStart      AuditAction = "build_start"
	AuditActionBuildComplete   AuditAction = "build_complete"
	AuditActionBuildCancel     AuditAction = "build_cancel"
	AuditActionQuotaCheck      AuditAction = "quota_check"
	AuditActionResourceAccess  AuditAction = "resource_access"
	AuditActionPolicyViolation AuditAction = "policy_violation"
	AuditActionStorageAccess   AuditAction = "storage_access"
	AuditActionNetworkAccess   AuditAction = "network_access"
)

// AuditResult represents the result of an audited action
type AuditResult string

const (
	AuditResultAllowed AuditResult = "allowed"
	AuditResultDenied  AuditResult = "denied"
	AuditResultError   AuditResult = "error"
)

// QuotaViolation represents a quota violation for reporting
type QuotaViolation struct {
	TenantID   string    `json:"tenant_id"`
	QuotaType  QuotaType `json:"quota_type"`
	Current    int64     `json:"current"`
	Limit      int64     `json:"limit"`
	Percentage float64   `json:"percentage"`
	Timestamp  time.Time `json:"timestamp"`
	Severity   string    `json:"severity"` // warning, critical
	Action     string    `json:"action"`   // throttled, blocked
	Duration   int64     `json:"duration"` // how long the violation lasted
}

// ResourceUsage represents detailed resource usage for a build
type ResourceUsage struct {
	BuildID   string        `json:"build_id"`
	TenantID  string        `json:"tenant_id"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`

	// CPU usage
	CPUUsagePercent float64       `json:"cpu_usage_percent"`
	CPUTimeTotal    time.Duration `json:"cpu_time_total"`
	CPUThrottleTime time.Duration `json:"cpu_throttle_time"`

	// Memory usage
	MemoryUsedBytes  int64 `json:"memory_used_bytes"`
	MemoryMaxBytes   int64 `json:"memory_max_bytes"`
	MemoryLimitBytes int64 `json:"memory_limit_bytes"`
	MemorySwapBytes  int64 `json:"memory_swap_bytes"`

	// Disk usage
	DiskReadBytes  int64 `json:"disk_read_bytes"`
	DiskWriteBytes int64 `json:"disk_write_bytes"`
	DiskUsedBytes  int64 `json:"disk_used_bytes"`
	DiskLimitBytes int64 `json:"disk_limit_bytes"`

	// Network usage
	NetworkRxBytes     int64 `json:"network_rx_bytes"`
	NetworkTxBytes     int64 `json:"network_tx_bytes"`
	NetworkConnections int32 `json:"network_connections"`

	// Process information
	ProcessCount        int32 `json:"process_count"`
	ThreadCount         int32 `json:"thread_count"`
	FileDescriptorCount int32 `json:"file_descriptor_count"`
}

// TenantMetrics represents aggregated metrics for a tenant
type TenantMetrics struct {
	TenantID  string    `json:"tenant_id"`
	Timestamp time.Time `json:"timestamp"`

	// Build metrics
	TotalBuilds      int64         `json:"total_builds"`
	SuccessfulBuilds int64         `json:"successful_builds"`
	FailedBuilds     int64         `json:"failed_builds"`
	CancelledBuilds  int64         `json:"cancelled_builds"`
	AvgBuildDuration time.Duration `json:"avg_build_duration"`

	// Resource metrics
	TotalCPUTime      time.Duration `json:"total_cpu_time"`
	TotalMemoryBytes  int64         `json:"total_memory_bytes"`
	TotalDiskBytes    int64         `json:"total_disk_bytes"`
	TotalNetworkBytes int64         `json:"total_network_bytes"`

	// Cost metrics (for billing)
	ComputeCost float64 `json:"compute_cost"`
	StorageCost float64 `json:"storage_cost"`
	NetworkCost float64 `json:"network_cost"`
	TotalCost   float64 `json:"total_cost"`

	// Quota violations
	QuotaViolations []QuotaViolation `json:"quota_violations"`
}
