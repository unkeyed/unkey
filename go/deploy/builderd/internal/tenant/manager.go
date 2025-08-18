package tenant

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/unkeyed/unkey/go/deploy/builderd/internal/config"
	builderv1 "github.com/unkeyed/unkey/go/gen/proto/deploy/builderd/v1"
)

// Manager handles tenant isolation, quotas, and resource management
type Manager struct {
	logger *slog.Logger
	config *config.Config

	// Active resource tracking
	activeBuilds   map[string]int32            // tenant_id -> active count
	dailyBuilds    map[string]map[string]int32 // tenant_id -> date -> count
	storageUsage   map[string]int64            // tenant_id -> bytes used
	computeMinutes map[string]map[string]int64 // tenant_id -> date -> minutes

	// Tenant configurations cache (using sync.Map for optimized reads)
	tenantConfigs sync.Map // map[string]*TenantConfig

	// Thread safety for other data structures
	mutex sync.RWMutex

	// Cleanup ticker
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// TenantConfig holds per-tenant configuration and limits
type TenantConfig struct {
	TenantID   string
	CustomerID string
	Tier       builderv1.TenantTier

	// Resource limits based on tier
	Limits TenantLimits

	// Network policies
	Network NetworkPolicy

	// Storage configuration
	Storage StorageConfig

	// Last updated timestamp
	UpdatedAt time.Time
}

// TenantLimits defines resource limits for a tenant
type TenantLimits struct {
	// Build limits
	MaxConcurrentBuilds int32
	MaxDailyBuilds      int32
	MaxBuildTimeMinutes int32

	// Resource limits per build
	MaxMemoryBytes int64
	MaxCPUCores    int32
	MaxDiskBytes   int64
	TimeoutSeconds int32

	// Storage limits
	MaxStorageBytes int64

	// Network limits
	AllowExternalNetwork bool
	AllowedRegistries    []string
	AllowedGitHosts      []string
}

// NetworkPolicy defines network access controls
type NetworkPolicy struct {
	AllowExternalNetwork bool
	AllowedRegistries    []string
	AllowedGitHosts      []string
	BlockedDomains       []string
	RequireVPN           bool
}

// StorageConfig defines storage isolation settings
type StorageConfig struct {
	IsolationEnabled   bool
	EncryptionEnabled  bool
	CompressionEnabled bool
	RetentionDays      int32
}

// NewManager creates a new tenant manager
func NewManager(logger *slog.Logger, cfg *config.Config) *Manager {
	manager := &Manager{ //nolint:exhaustruct // tenantConfigs is sync.Map (zero-value), mutex is sync.RWMutex (zero-value), cleanupTicker set below
		logger:         logger,
		config:         cfg,
		activeBuilds:   make(map[string]int32),
		dailyBuilds:    make(map[string]map[string]int32),
		storageUsage:   make(map[string]int64),
		computeMinutes: make(map[string]map[string]int64),
		// tenantConfigs is a sync.Map, no initialization needed
		stopCleanup: make(chan struct{}),
	}

	// Start cleanup ticker for daily counters
	manager.cleanupTicker = time.NewTicker(1 * time.Hour)
	go manager.startCleanup()

	logger.InfoContext(context.Background(), "tenant manager initialized")
	return manager
}

// GetTenantConfig retrieves or creates tenant configuration
func (m *Manager) GetTenantConfig(ctx context.Context, tenantID string, tier builderv1.TenantTier) (*TenantConfig, error) {
	// Fast path: check if tenant config exists (lock-free read)
	if value, exists := m.tenantConfigs.Load(tenantID); exists {
		config, _ := value.(*TenantConfig)
		return config, nil
	}

	// Create new tenant config - no manual locking needed with sync.Map

	config := &TenantConfig{ //nolint:exhaustruct // CustomerID is optional and not required for basic tenant configuration
		TenantID:  tenantID,
		Tier:      tier,
		Limits:    m.getTierLimits(tier),
		Network:   m.getNetworkPolicy(tier),
		Storage:   m.getStorageConfig(tier),
		UpdatedAt: time.Now(),
	}

	// Use LoadOrStore to handle race conditions atomically
	if actual, loaded := m.tenantConfigs.LoadOrStore(tenantID, config); loaded {
		// Another goroutine created the config first, use that one
		actualConfig, _ := actual.(*TenantConfig)
		return actualConfig, nil
	}

	m.logger.InfoContext(ctx, "created tenant configuration",
		slog.String("tenant_id", tenantID),
		slog.String("tier", tier.String()),
		slog.Int64("max_concurrent_builds", int64(config.Limits.MaxConcurrentBuilds)),
		slog.Int64("max_daily_builds", int64(config.Limits.MaxDailyBuilds)),
	)

	return config, nil
}

// CheckBuildQuotas validates if a tenant can start a new build
func (m *Manager) CheckBuildQuotas(ctx context.Context, tenantID string, tier builderv1.TenantTier) error {
	config, err := m.GetTenantConfig(ctx, tenantID, tier)
	if err != nil {
		return fmt.Errorf("failed to get tenant config: %w", err)
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Check concurrent builds limit
	activeBuildCount := m.activeBuilds[tenantID]
	if activeBuildCount >= config.Limits.MaxConcurrentBuilds {
		return &QuotaError{
			Type:     QuotaTypeConcurrentBuilds,
			TenantID: tenantID,
			Current:  int64(activeBuildCount),
			Limit:    int64(config.Limits.MaxConcurrentBuilds),
			Message:  fmt.Sprintf("concurrent build limit exceeded: %d/%d", activeBuildCount, config.Limits.MaxConcurrentBuilds),
		}
	}

	// Check daily builds limit
	today := time.Now().Format("2006-01-02")
	dailyCount := int32(0)
	if tenantDaily, exists := m.dailyBuilds[tenantID]; exists {
		dailyCount = tenantDaily[today]
	}

	if dailyCount >= config.Limits.MaxDailyBuilds {
		return &QuotaError{
			Type:     QuotaTypeDailyBuilds,
			TenantID: tenantID,
			Current:  int64(dailyCount),
			Limit:    int64(config.Limits.MaxDailyBuilds),
			Message:  fmt.Sprintf("daily build limit exceeded: %d/%d", dailyCount, config.Limits.MaxDailyBuilds),
		}
	}

	// Check storage quota
	storageUsed := m.storageUsage[tenantID]
	if storageUsed >= config.Limits.MaxStorageBytes {
		return &QuotaError{
			Type:     QuotaTypeStorage,
			TenantID: tenantID,
			Current:  storageUsed,
			Limit:    config.Limits.MaxStorageBytes,
			Message:  fmt.Sprintf("storage quota exceeded: %d/%d bytes", storageUsed, config.Limits.MaxStorageBytes),
		}
	}

	return nil
}

// ReserveBuildSlot reserves a build slot for a tenant
func (m *Manager) ReserveBuildSlot(ctx context.Context, tenantID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Increment active builds
	m.activeBuilds[tenantID]++

	// Increment daily builds
	today := time.Now().Format("2006-01-02")
	if m.dailyBuilds[tenantID] == nil {
		m.dailyBuilds[tenantID] = make(map[string]int32)
	}
	m.dailyBuilds[tenantID][today]++

	m.logger.DebugContext(ctx, "reserved build slot",
		slog.String("tenant_id", tenantID),
		slog.Int64("active_builds", int64(m.activeBuilds[tenantID])),
		slog.Int64("daily_builds", int64(m.dailyBuilds[tenantID][today])),
	)

	return nil
}

// ReleaseBuildSlot releases a build slot for a tenant
func (m *Manager) ReleaseBuildSlot(ctx context.Context, tenantID string, buildDurationMinutes int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Decrement active builds
	if m.activeBuilds[tenantID] > 0 {
		m.activeBuilds[tenantID]--
	}

	// Track compute minutes
	today := time.Now().Format("2006-01-02")
	if m.computeMinutes[tenantID] == nil {
		m.computeMinutes[tenantID] = make(map[string]int64)
	}
	m.computeMinutes[tenantID][today] += buildDurationMinutes

	m.logger.DebugContext(ctx, "released build slot",
		slog.String("tenant_id", tenantID),
		slog.Int64("active_builds", int64(m.activeBuilds[tenantID])),
		slog.Int64("build_duration_minutes", buildDurationMinutes),
	)
}

// UpdateStorageUsage updates storage usage for a tenant
func (m *Manager) UpdateStorageUsage(ctx context.Context, tenantID string, deltaBytes int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.storageUsage[tenantID] += deltaBytes
	if m.storageUsage[tenantID] < 0 {
		m.storageUsage[tenantID] = 0
	}

	m.logger.DebugContext(ctx, "updated storage usage",
		slog.String("tenant_id", tenantID),
		slog.Int64("delta_bytes", deltaBytes),
		slog.Int64("total_bytes", m.storageUsage[tenantID]),
	)
}

// GetUsageStats returns current usage statistics for a tenant
func (m *Manager) GetUsageStats(ctx context.Context, tenantID string) *UsageStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	today := time.Now().Format("2006-01-02")

	stats := &UsageStats{ //nolint:exhaustruct // DailyBuildsUsed and ComputeMinutesUsed are populated conditionally below based on existence
		TenantID:         tenantID,
		ActiveBuilds:     m.activeBuilds[tenantID],
		StorageBytesUsed: m.storageUsage[tenantID],
		Timestamp:        time.Now(),
	}

	if tenantDaily, exists := m.dailyBuilds[tenantID]; exists {
		stats.DailyBuildsUsed = tenantDaily[today]
	}

	if tenantCompute, exists := m.computeMinutes[tenantID]; exists {
		stats.ComputeMinutesUsed = tenantCompute[today]
	}

	return stats
}

// getTierLimits returns resource limits based on tenant tier
func (m *Manager) getTierLimits(tier builderv1.TenantTier) TenantLimits {
	switch tier {
	case builderv1.TenantTier_TENANT_TIER_UNSPECIFIED:
		// Default to free tier limits for unspecified
		return m.getTierLimits(builderv1.TenantTier_TENANT_TIER_FREE)
	case builderv1.TenantTier_TENANT_TIER_FREE:
		return TenantLimits{
			MaxConcurrentBuilds:  1,
			MaxDailyBuilds:       5,
			MaxBuildTimeMinutes:  5,
			MaxMemoryBytes:       512 * 1024 * 1024, // 512MB
			MaxCPUCores:          1,
			MaxDiskBytes:         1024 * 1024 * 1024, // 1GB
			TimeoutSeconds:       300,                // 5 min
			MaxStorageBytes:      1024 * 1024 * 1024, // 1GB
			AllowExternalNetwork: false,
			AllowedRegistries:    []string{"docker.io", "ghcr.io"},
			AllowedGitHosts:      []string{"github.com"},
		}
	case builderv1.TenantTier_TENANT_TIER_PRO:
		return TenantLimits{
			MaxConcurrentBuilds:  3,
			MaxDailyBuilds:       100,
			MaxBuildTimeMinutes:  15,
			MaxMemoryBytes:       2 * 1024 * 1024 * 1024, // 2GB
			MaxCPUCores:          2,
			MaxDiskBytes:         10 * 1024 * 1024 * 1024, // 10GB
			TimeoutSeconds:       900,                     // 15 min
			MaxStorageBytes:      10 * 1024 * 1024 * 1024, // 10GB
			AllowExternalNetwork: true,
			AllowedRegistries:    []string{"*"},
			AllowedGitHosts:      []string{"*"},
		}
	case builderv1.TenantTier_TENANT_TIER_ENTERPRISE:
		return TenantLimits{
			MaxConcurrentBuilds:  10,
			MaxDailyBuilds:       1000,
			MaxBuildTimeMinutes:  30,
			MaxMemoryBytes:       8 * 1024 * 1024 * 1024, // 8GB
			MaxCPUCores:          4,
			MaxDiskBytes:         100 * 1024 * 1024 * 1024, // 100GB
			TimeoutSeconds:       1800,                     // 30 min
			MaxStorageBytes:      100 * 1024 * 1024 * 1024, // 100GB
			AllowExternalNetwork: true,
			AllowedRegistries:    []string{"*"},
			AllowedGitHosts:      []string{"*"},
		}
	case builderv1.TenantTier_TENANT_TIER_DEDICATED:
		return TenantLimits{
			MaxConcurrentBuilds:  50,
			MaxDailyBuilds:       10000,
			MaxBuildTimeMinutes:  60,
			MaxMemoryBytes:       32 * 1024 * 1024 * 1024, // 32GB
			MaxCPUCores:          16,
			MaxDiskBytes:         1024 * 1024 * 1024 * 1024, // 1TB
			TimeoutSeconds:       3600,                      // 60 min
			MaxStorageBytes:      1024 * 1024 * 1024 * 1024, // 1TB
			AllowExternalNetwork: true,
			AllowedRegistries:    []string{"*"},
			AllowedGitHosts:      []string{"*"},
		}
	default:
		// Default to free tier limits
		return m.getTierLimits(builderv1.TenantTier_TENANT_TIER_FREE)
	}
}

// getNetworkPolicy returns network policy based on tenant tier
func (m *Manager) getNetworkPolicy(tier builderv1.TenantTier) NetworkPolicy {
	switch tier {
	case builderv1.TenantTier_TENANT_TIER_UNSPECIFIED:
		// Default to free tier policy for unspecified
		return m.getNetworkPolicy(builderv1.TenantTier_TENANT_TIER_FREE)
	case builderv1.TenantTier_TENANT_TIER_FREE:
		return NetworkPolicy{
			AllowExternalNetwork: false,
			AllowedRegistries:    []string{"docker.io", "ghcr.io"},
			AllowedGitHosts:      []string{"github.com", "gitlab.com"},
			BlockedDomains:       []string{},
			RequireVPN:           false,
		}
	case builderv1.TenantTier_TENANT_TIER_PRO, builderv1.TenantTier_TENANT_TIER_ENTERPRISE, builderv1.TenantTier_TENANT_TIER_DEDICATED:
		return NetworkPolicy{
			AllowExternalNetwork: true,
			AllowedRegistries:    []string{"*"}, // All registries
			AllowedGitHosts:      []string{"*"}, // All git hosts
			BlockedDomains:       []string{},
			RequireVPN:           false,
		}
	default:
		return m.getNetworkPolicy(builderv1.TenantTier_TENANT_TIER_FREE)
	}
}

// getStorageConfig returns storage configuration based on tenant tier
func (m *Manager) getStorageConfig(tier builderv1.TenantTier) StorageConfig {
	switch tier {
	case builderv1.TenantTier_TENANT_TIER_UNSPECIFIED:
		// Default to free tier storage config for unspecified
		return m.getStorageConfig(builderv1.TenantTier_TENANT_TIER_FREE)
	case builderv1.TenantTier_TENANT_TIER_FREE, builderv1.TenantTier_TENANT_TIER_PRO:
		return StorageConfig{
			IsolationEnabled:   true,
			EncryptionEnabled:  false,
			CompressionEnabled: true,
			RetentionDays:      30,
		}
	case builderv1.TenantTier_TENANT_TIER_ENTERPRISE, builderv1.TenantTier_TENANT_TIER_DEDICATED:
		return StorageConfig{
			IsolationEnabled:   true,
			EncryptionEnabled:  true,
			CompressionEnabled: true,
			RetentionDays:      90,
		}
	default:
		return m.getStorageConfig(builderv1.TenantTier_TENANT_TIER_FREE)
	}
}

// startCleanup runs periodic cleanup of old data
func (m *Manager) startCleanup() {
	for {
		select {
		case <-m.cleanupTicker.C:
			m.cleanupOldData()
		case <-m.stopCleanup:
			return
		}
	}
}

// cleanupOldData removes old daily counters and unused tenant configs
func (m *Manager) cleanupOldData() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	cutoff := time.Now().AddDate(0, 0, -7).Format("2006-01-02") // Keep 7 days

	// Cleanup old daily build counters
	for tenantID, dailyMap := range m.dailyBuilds {
		for date := range dailyMap {
			if date < cutoff {
				delete(dailyMap, date)
			}
		}
		if len(dailyMap) == 0 {
			delete(m.dailyBuilds, tenantID)
		}
	}

	// Cleanup old compute minute counters
	for tenantID, computeMap := range m.computeMinutes {
		for date := range computeMap {
			if date < cutoff {
				delete(computeMap, date)
			}
		}
		if len(computeMap) == 0 {
			delete(m.computeMinutes, tenantID)
		}
	}

	m.logger.DebugContext(context.Background(), "cleaned up old tenant data")
}

// Shutdown gracefully shuts down the tenant manager
func (m *Manager) Shutdown() {
	if m.cleanupTicker != nil {
		m.cleanupTicker.Stop()
	}
	close(m.stopCleanup)
	m.logger.InfoContext(context.Background(), "tenant manager shutdown")
}
