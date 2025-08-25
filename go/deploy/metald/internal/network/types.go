package network

import (
	"fmt"
	"net"
	"time"
)

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

	// Optional fields for advanced configurations
	VLANID      int     `json:"vlan_id,omitempty"`
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

// GenerateCloudInitNetwork generates cloud-init network configuration
func (n *VMNetwork) GenerateCloudInitNetwork() map[string]interface{} {
	// Generate network configuration for cloud-init
	config := map[string]interface{}{
		"version": 2,
		"ethernets": map[string]interface{}{
			"eth0": map[string]interface{}{
				"match": map[string]interface{}{
					"macaddress": n.MacAddress,
				},
				"addresses": []string{
					n.IPAddress.String() + "/24",
				},
				"gateway4": n.Gateway.String(),
				"nameservers": map[string]interface{}{
					"addresses": n.DNSServers,
				},
			},
		},
	}

	return config
}

// GenerateNetworkMetadata generates metadata for the VM
func (n *VMNetwork) GenerateNetworkMetadata() map[string]string {
	metadata := map[string]string{
		"local-ipv4":      n.IPAddress.String(),
		"mac":             n.MacAddress,
		"gateway":         n.Gateway.String(),
		"netmask":         n.Netmask.String(),
		"dns-nameservers": n.DNSServers[0],
	}

	if len(n.DNSServers) > 1 {
		metadata["dns-nameservers-secondary"] = n.DNSServers[1]
	}

	return metadata
}

// KernelCmdlineArgs returns kernel command line arguments for network configuration
func (n *VMNetwork) KernelCmdlineArgs() string {
	// Format: ip=<client-ip>:<server-ip>:<gw-ip>:<netmask>:<hostname>:<device>:<autoconf>
	// Example: ip=10.100.1.2::10.100.0.1:255.255.255.0:vm::off
	return fmt.Sprintf("ip=%s::%s:%s:vm::off",
		n.IPAddress.String(),
		n.Gateway.String(),
		n.Netmask.String(),
	)
}
