package network

import (
	"encoding/json"
	"fmt"
	"net"
)

// GenerateAvailableIPs generates a JSON array of available IP addresses from a CIDR network
// It excludes the network address (first IP), gateway (second IP), and broadcast (last IP)
func GenerateAvailableIPs(cidr string) (string, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", fmt.Errorf("invalid CIDR: %w", err)
	}

	// Calculate the number of hosts
	ones, bits := ipnet.Mask.Size()
	hostBits := bits - ones
	numHosts := (1 << hostBits) - 2 // Exclude network and broadcast

	if numHosts < 1 {
		return "[]", nil // No usable IPs in this network
	}

	// Generate IPs
	ips := make([]string, 0, numHosts-1) // -1 to also exclude gateway

	// Start from the third IP (skip network and gateway)
	ip := ipnet.IP.Mask(ipnet.Mask)
	incIP(ip) // Skip network address
	incIP(ip) // Skip gateway address

	// Generate IPs until we reach the broadcast address
	broadcast := getBroadcast(ipnet)
	for i := 0; i < numHosts-1 && !ip.Equal(broadcast); i++ {
		ips = append(ips, ip.String())
		incIP(ip)
	}

	// Convert to JSON array
	jsonBytes, err := json.Marshal(ips)
	if err != nil {
		return "", fmt.Errorf("failed to marshal IPs to JSON: %w", err)
	}

	return string(jsonBytes), nil
}

// incIP increments an IP address by 1
func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// getBroadcast calculates the broadcast address for a network
func getBroadcast(ipnet *net.IPNet) net.IP {
	ip := ipnet.IP.To4()
	mask := ipnet.Mask
	broadcast := make(net.IP, len(ip))

	for i := range ip {
		broadcast[i] = ip[i] | ^mask[i]
	}

	return broadcast
}
