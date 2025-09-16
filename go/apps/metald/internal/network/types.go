package network

import (
	"log/slog"
	"net"
	"sync"
	"time"
)

// BridgeManager manages workspace allocation across multiple bridges
type BridgeManager struct {
	bridgeCount  int                             // 8 or 32 bridges
	bridgePrefix string                          // "br-vms" -> br-vms-0, br-vms-1, etc.
	workspaces   map[string]*WorkspaceAllocation // workspace_id -> allocation
	bridgeUsage  map[int]map[string]bool         // bridge_num -> workspace_id -> exists
	mu           sync.RWMutex
	statePath    string       // Path to state persistence file
	logger       *slog.Logger // Structured logger for state operations
}

// BridgeState represents the serializable state for persistence
type BridgeState struct {
	Workspaces  map[string]*WorkspaceAllocation `json:"workspaces"`
	BridgeUsage map[int]map[string]bool         `json:"bridge_usage"`
	LastSaved   time.Time                       `json:"last_saved"`
	Checksum    string                          `json:"checksum"` // SHA256 checksum for integrity validation
}

type MultiBridgeManager struct {
	bridgeCount    int                             // 8 or 32 bridges
	bridgePrefix   string                          // "br-vms" -> br-vms-0, br-vms-1, etc.
	workspaces     map[string]*WorkspaceAllocation // workspace_id -> allocation
	bridgeUsage    map[int]map[string]bool         // bridge_num -> workspace_id -> exists
	mu             sync.RWMutex
	vlanRangeStart int          // Starting VLAN ID (100)
	vlanRangeEnd   int          // Ending VLAN ID (4000)
	statePath      string       // Path to state persistence file
	logger         *slog.Logger // Structured logger for state operations
}

// WorkspaceAllocation represents a workspace's network allocation
type WorkspaceAllocation struct {
	WorkspaceID  string `json:"workspace_id"`
	BridgeNumber int    `json:"bridge_number"` // 0-31
	BridgeName   string `json:"bridge_name"`   // br-vms-N
	CreatedAt    string `json:"created_at"`
	VMCount      int    `json:"vm_count"` // Track VM count for IP allocation
}

// VMNetwork contains network configuration for a VM
type VMNetwork struct {
	VMID        string     `json:"vm_id"`
	NetworkID   string     `json:"network_id"`   // AIDEV-NOTE: Internal 8-char ID for network device naming
	WorkspaceID string     `json:"workspace_id"` // AIDEV-NOTE: Track workspace for proper IP release
	Namespace   string     `json:"namespace"`
	TapDevice   string     `json:"tap_device"`
	IPAddress   net.IP     `json:"ip_address"`
	Netmask     net.IPMask `json:"netmask"`
	Gateway     net.IP     `json:"gateway"`
	MacAddress  string     `json:"mac_address"`
	DNSServers  []string   `json:"dns_servers"`
	CreatedAt   time.Time  `json:"created_at"`

	// Optional fields for advanced configuration
	IPv6Address net.IP  `json:"ipv6_address,omitempty"`
	Routes      []Route `json:"routes,omitempty"`
}

// Route represents a network route
type Route struct {
	Destination *net.IPNet `json:"destination"`
	Gateway     net.IP     `json:"gateway"`
	Metric      int        `json:"metric"`
}

// NetworkStats contains network interface statistics
type NetworkStats struct {
	RxBytes   uint64 `json:"rx_bytes"`
	TxBytes   uint64 `json:"tx_bytes"`
	RxPackets uint64 `json:"rx_packets"`
	TxPackets uint64 `json:"tx_packets"`
	RxDropped uint64 `json:"rx_dropped"`
	TxDropped uint64 `json:"tx_dropped"`
	RxErrors  uint64 `json:"rx_errors"`
	TxErrors  uint64 `json:"tx_errors"`
}

// NetworkPolicy defines network access rules for a VM
type NetworkPolicy struct {
	VMID          string         `json:"vm_id"`
	CustomerID    string         `json:"customer_id"`
	Rules         []FirewallRule `json:"rules"`
	DefaultAction string         `json:"default_action"` // "allow" or "deny"
}

// FirewallRule defines a single firewall rule
type FirewallRule struct {
	Name        string `json:"name"`
	Direction   string `json:"direction"` // "ingress" or "egress"
	Protocol    string `json:"protocol"`  // "tcp", "udp", "icmp", or ""
	Port        int    `json:"port,omitempty"`
	PortRange   string `json:"port_range,omitempty"`  // e.g., "8080-8090"
	Source      string `json:"source"`                // CIDR or "any"
	Destination string `json:"destination,omitempty"` // CIDR or "any"
	Action      string `json:"action"`                // "allow" or "deny"
	Priority    int    `json:"priority"`              // Lower number = higher priority
}
