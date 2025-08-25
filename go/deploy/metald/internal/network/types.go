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
// AIDEV-NOTE: Updated to use correct Firecracker format: ip=G::T:GM::GI:off
// where G=guest IP, T=TAP IP, GM=guest mask, GI=guest interface
func (n *VMNetwork) KernelCmdlineArgs() string {
	if n.IPAddress == nil {
		return ""
	}

	// Calculate the actual host-side veth IP for this /29 subnet
	// The host veth gets the first IP in the VM's /29 subnet range
	tapIP := calculateVethHostIP(n.IPAddress)

	// Convert netmask to dotted decimal format
	netmaskStr := n.formatNetmask()

	// Guest interface name (typically eth0 for the first interface)
	guestInterface := "eth0"

	// Format: ip=G::T:GM::GI:off
	// G = Guest IP, T = TAP IP, GM = Guest Mask, GI = Guest Interface
	return fmt.Sprintf("ip=%s::%s:%s:%s:off",
		n.IPAddress.String(),
		tapIP,
		netmaskStr,
		guestInterface,
	)
}

// formatNetmask converts the netmask to dotted decimal format
func (n *VMNetwork) formatNetmask() string {
	if n.Netmask == nil {
		// Default to /24 if no netmask specified
		return "255.255.255.0"
	}

	// Handle net.IPMask directly
	if len(n.Netmask) == 4 {
		// IPv4 netmask - convert directly to dotted decimal
		return fmt.Sprintf("%d.%d.%d.%d",
			n.Netmask[0], n.Netmask[1], n.Netmask[2], n.Netmask[3])
	}

	// Handle IPv6-style netmask (16 bytes) - extract IPv4 part
	if len(n.Netmask) == 16 {
		// IPv4-mapped in IPv6 format, take last 4 bytes
		return fmt.Sprintf("%d.%d.%d.%d",
			n.Netmask[12], n.Netmask[13], n.Netmask[14], n.Netmask[15])
	}

	// Try converting to net.IP as fallback
	mask := net.IP(n.Netmask)
	if mask != nil {
		maskStr := mask.String()
		// Validate it looks like a dotted decimal IP
		if len(maskStr) > 0 && maskStr != "<nil>" {
			return maskStr
		}
	}

	// Final fallback
	return "255.255.255.0"
}
