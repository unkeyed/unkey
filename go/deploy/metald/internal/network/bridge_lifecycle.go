package network

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/unkeyed/unkey/go/deploy/metald/internal/config"
)

// Config holds network configuration
type Config struct {
	BridgeName      string // Default: "br-vms"
	BridgeIP        string // Default: "172.31.0.1/19"
	VMSubnet        string // Default: "172.31.0.0/19"
	EnableIPv6      bool
	DNSServers      []string // Default: ["8.8.8.8", "8.8.4.4"]
	EnableRateLimit bool
	RateLimitMbps   int // Per VM rate limit in Mbps

	// Port allocation configuration
	PortRangeMin int // Default: 32768
	PortRangeMax int // Default: 65535
}

// DefaultConfig returns default network configuration
func DefaultConfig() *Config {
	return &Config{ //nolint:exhaustruct // EnableIPv6 field uses zero value (false) which is appropriate for default config
		BridgeName:      "br-vms",
		BridgeIP:        "172.31.0.1/19",
		VMSubnet:        "172.31.0.0/19",
		DNSServers:      []string{"8.8.8.8", "8.8.4.4"},
		EnableRateLimit: true,
		RateLimitMbps:   100,   // 100 Mbps default
		PortRangeMin:    32768, // Ephemeral port range start
		PortRangeMax:    65535, // Ephemeral port range end
	}
}

// Manager handles VM networking
type Manager struct {
	logger        *slog.Logger
	config        *Config
	allocator     *IPAllocator
	portAllocator *PortAllocator
	idGen         *IDGenerator
	mu            sync.RWMutex
	vmNetworks    map[string]*VMNetwork

	// Runtime state (hostProtection removed - managed externally)

	// Multi-tenant workspace management
	multiBridgeManager *MultiBridgeManager
	metrics            *NetworkMetrics

	// AIDEV-NOTE: CRITICAL FIX - Bridge state synchronization
	// bridgeMu protects bridgeCreated and bridgeInitTime to prevent race conditions
	// during bridge verification and creation operations
	bridgeMu       sync.RWMutex
	bridgeCreated  bool
	bridgeInitTime time.Time

	iptablesRules []string
}

// NewManager creates a new network manager
func NewManager(logger *slog.Logger, netConfig *Config, mainConfig *config.NetworkConfig) (*Manager, error) {
	if netConfig == nil {
		netConfig = DefaultConfig()
	}

	logger = logger.With("component", "network-manager")
	logger.Info("creating network manager",
		slog.String("bridge_name", netConfig.BridgeName),
		slog.String("bridge_ip", netConfig.BridgeIP),
		slog.String("vm_subnet", netConfig.VMSubnet),
		slog.Bool("host_protection", mainConfig.EnableHostProtection),
	)

	_, subnet, err := net.ParseCIDR(netConfig.VMSubnet)
	if err != nil {
		return nil, fmt.Errorf("invalid subnet: %w", err)
	}

	// Initialize network metrics
	networkMetrics, err := NewNetworkMetrics(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create network metrics: %w", err)
	}

	m := &Manager{ //nolint:exhaustruct // mu, bridgeCreated, and iptablesRules fields use appropriate zero values
		logger:             logger,
		config:             netConfig,
		allocator:          NewIPAllocator(subnet),
		portAllocator:      NewPortAllocator(netConfig.PortRangeMin, netConfig.PortRangeMax),
		idGen:              NewIDGenerator(),
		metrics:            networkMetrics,
		vmNetworks:         make(map[string]*VMNetwork),
		multiBridgeManager: NewMultiBridgeManager(mainConfig.BridgeCount, "br-tenant", logger),
	}

	// Set bridge max VMs based on configuration
	m.metrics.SetBridgeMaxVMs(netConfig.BridgeName, int64(mainConfig.MaxVMsPerBridge))

	// Log current network state before initialization
	m.logNetworkState("before initialization")

	// Initialize host networking
	if err := m.initializeHost(); err != nil {
		m.logger.Error("failed to initialize host networking",
			slog.String("error", err.Error()),
		)
		m.logNetworkState("after failed initialization")
		return nil, fmt.Errorf("failed to initialize host networking: %w", err)
	}

	// Log network state after verification
	m.logNetworkState("after successful bridge verification")

	return m, nil
}

// initializeHost sets up the host networking infrastructure
func (m *Manager) initializeHost() error {
	m.logger.Info("starting host network initialization")

	// Enable IP forwarding by writing directly to proc filesystem
	m.logger.Info("enabling IP forwarding")
	err := os.WriteFile("/proc/sys/net/ipv4/ip_forward", []byte("1"), 0644)
	if err != nil {
		m.logger.Error("failed to enable IP forwarding",
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to enable IP forwarding: %w", err)
	}

	// Make it persistent across reboots
	// AIDEV-NOTE: Creates sysctl config to persist IP forwarding
	sysctlConfig := []byte("# Enable IP forwarding for metald VM networking\nnet.ipv4.ip_forward = 1\n")
	sysctlPath := "/etc/sysctl.d/99-metald.conf"

	if err := os.WriteFile(sysctlPath, sysctlConfig, 0600); err != nil {
		m.logger.Warn("failed to create persistent sysctl config",
			slog.String("path", sysctlPath),
			slog.String("error", err.Error()),
		)
		// Not fatal - IP forwarding is enabled for this session
	}

	m.logger.Info("IP forwarding enabled successfully")

	// Create bridge if it doesn't exist
	if err := m.ensureBridge(); err != nil {
		return fmt.Errorf("failed to create bridge: %w", err)
	}

	// Setup NAT rules (best effort - may fail without root or if already configured)
	m.logNetworkState("before NAT setup")
	if err := m.setupNAT(); err != nil {
		m.logger.Warn("failed to setup NAT (may already be configured)",
			slog.String("error", err.Error()),
		)
		m.logNetworkState("after failed NAT setup")
		// Continue anyway - NAT might already be set up
	} else {
		m.logNetworkState("after successful NAT setup")
	}

	m.logger.Info("host networking initialized",
		slog.String("bridge", m.config.BridgeName),
		slog.String("subnet", m.config.VMSubnet),
	)

	return nil
}

// Shutdown cleans up all networking resources
func (m *Manager) Shutdown(ctx context.Context) error {
	m.logger.InfoContext(ctx, "shutting down network manager")
	m.logNetworkState("before shutdown")

	// AIDEV-NOTE: CRITICAL FIX - Preserve IP allocation state during service restarts
	// Only clean up physical resources, not network state, to prevent IP allocation corruption
	vmCount := len(m.vmNetworks)
	m.logger.InfoContext(ctx, "preserving VM network state during shutdown",
		slog.Int("count", vmCount),
		slog.String("reason", "service restart should not affect IP allocation"),
	)

	// Note: We intentionally do NOT call DeleteVMNetwork here during shutdown
	// DeleteVMNetwork would decrement VMCount in MultiBridgeManager, corrupting IP allocation state
	// Physical cleanup (namespaces, interfaces) will be handled by the OS or next service start

	// Clean up iptables rules
	m.logger.InfoContext(ctx, "cleaning up iptables rules",
		slog.Int("rule_count", len(m.iptablesRules)),
	)
	m.cleanupIPTables()

	// AIDEV-NOTE: We intentionally keep the bridge to avoid network disruption
	// Deleting the bridge can cause host network issues if there are dependencies
	m.logger.InfoContext(ctx, "keeping bridge intact to avoid network disruption",
		slog.String("bridge", m.config.BridgeName),
		slog.Bool("bridge_created", m.bridgeCreated),
	)

	m.logNetworkState("after shutdown")
	m.logger.InfoContext(ctx, "network manager shutdown complete")

	return nil
}

// cleanupIPTables removes iptables rules that were added by this manager
func (m *Manager) cleanupIPTables() {
	for _, ruleStr := range m.iptablesRules {
		// Convert ADD rule to DELETE rule
		deleteRule := strings.Replace(ruleStr, "-A ", "-D ", 1)
		deleteRule = strings.Replace(deleteRule, "-t nat -A ", "-t nat -D ", 1)

		m.logger.Info("removing iptables rule",
			slog.String("rule", deleteRule),
		)

		cmd := exec.Command("bash", "-c", "iptables "+deleteRule)
		if output, err := cmd.CombinedOutput(); err != nil {
			m.logger.Warn("failed to remove iptables rule",
				slog.String("rule", deleteRule),
				slog.String("error", err.Error()),
				slog.String("output", string(output)),
			)
		}
	}
	m.iptablesRules = nil
}

// GetBridgeCapacityStatus returns current bridge capacity and utilization
func (m *Manager) GetBridgeCapacityStatus() *BridgeCapacityStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bridgeStats := m.metrics.GetBridgeStats()
	alerts := m.metrics.GetBridgeCapacityAlerts()

	// Calculate overall statistics
	totalVMs := int64(0)
	totalCapacity := int64(0)
	bridgeCount := len(bridgeStats)

	bridgeDetails := make([]BridgeDetails, 0, bridgeCount)
	for _, stats := range bridgeStats {
		totalVMs += stats.VMCount
		totalCapacity += stats.MaxVMs

		utilization := float64(stats.VMCount) / float64(stats.MaxVMs)
		bridgeDetails = append(bridgeDetails, BridgeDetails{
			Name:         stats.BridgeName,
			VMCount:      stats.VMCount,
			MaxVMs:       stats.MaxVMs,
			Utilization:  utilization,
			IsHealthy:    stats.IsHealthy,
			CreatedAt:    stats.CreatedAt,
			LastActivity: stats.LastActivity,
		})
	}

	overallUtilization := float64(0)
	if totalCapacity > 0 {
		overallUtilization = float64(totalVMs) / float64(totalCapacity)
	}

	return &BridgeCapacityStatus{
		TotalVMs:           totalVMs,
		TotalCapacity:      totalCapacity,
		OverallUtilization: overallUtilization,
		BridgeCount:        int64(bridgeCount),
		Bridges:            bridgeDetails,
		Alerts:             alerts,
		Timestamp:          time.Now(),
	}
}

// GetNetworkMetrics returns the network metrics instance for external access
func (m *Manager) GetNetworkMetrics() *NetworkMetrics {
	return m.metrics
}

// BridgeCapacityStatus provides comprehensive bridge capacity information
type BridgeCapacityStatus struct {
	TotalVMs           int64                 `json:"total_vms"`
	TotalCapacity      int64                 `json:"total_capacity"`
	OverallUtilization float64               `json:"overall_utilization"`
	BridgeCount        int64                 `json:"bridge_count"`
	Bridges            []BridgeDetails       `json:"bridges"`
	Alerts             []BridgeCapacityAlert `json:"alerts"`
	Timestamp          time.Time             `json:"timestamp"`
}

// BridgeDetails provides detailed information about a specific bridge
type BridgeDetails struct {
	Name         string    `json:"name"`
	VMCount      int64     `json:"vm_count"`
	MaxVMs       int64     `json:"max_vms"`
	Utilization  float64   `json:"utilization"`
	IsHealthy    bool      `json:"is_healthy"`
	CreatedAt    time.Time `json:"created_at"`
	LastActivity time.Time `json:"last_activity"`
}
