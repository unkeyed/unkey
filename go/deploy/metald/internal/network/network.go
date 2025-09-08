package network

import (
	"fmt"
	"hash/fnv"
)

const (
	// BaseNetwork is the "root" network we're partitioning
	BaseNetwork = "172.16.0.0/12"

	// SubnetPrefix is the size of each subnet
	SubnetPrefix = 28

	// BasePrefix is the size of the base network
	BasePrefix = 12

	// TotalSubnets is the total number of /28 subnets in a /12
	TotalSubnets = 1 << (SubnetPrefix - BasePrefix) // 65536 = 1 << (28 - 12)

	// IPsPerSubnet is the number of IPs in each /28 subnet
	IPsPerSubnet = 1 << (32 - SubnetPrefix) // 16
)

// SubnetInfo contains all the network information for a subnet
type SubnetInfo struct {
	Index       uint32 // 0-based index (0-65535)
	Network     string // CIDR notation (e.g., "172.16.0.0/28")
	Gateway     string // Gateway IP (e.g., "172.16.0.1")
	Broadcast   string // Broadcast IP (e.g., "172.16.0.15")
	UsableRange string // Usable IP range (e.g., "172.16.0.2-172.16.0.14")
	UsableIPs   int    // Number of usable IPs (13 for /28)
}

// CalculateIndex returns a subnet index (0-65535) for a given workspace ID
func CalculateIndex(identifier string) uint32 {
	hash := fnv.New32a()
	hash.Write([]byte(identifier))
	return hash.Sum32() & 0xFFFF // Masks to keep only lower 16 bits
}

// CalculateIndexOneBased returns a subnet index (1-65536) for a given workspace ID
func CalculateIndexOneBased(identifier string) uint32 {
	return CalculateIndex(identifier) + 1
}

// GetSubnetInfo returns complete subnet information for a workspace ID
func GetSubnetInfo(identifier string) SubnetInfo {
	index := CalculateIndex(identifier)
	return GetSubnetInfoByIndex(index)
}

// GetSubnetInfoByIndex returns complete subnet information for a given index
func GetSubnetInfoByIndex(index uint32) SubnetInfo {
	if index >= TotalSubnets {
		panic(fmt.Sprintf("index %d exceeds maximum subnet count %d", index, TotalSubnets))
	}

	// Calculate the base IP for this subnet
	subnetOffset := index * IPsPerSubnet

	// Calculate octets for 172.16.0.0/12 base
	octet4 := subnetOffset % 256
	octet3 := (subnetOffset / 256) % 256
	octet2 := 16 + (subnetOffset / 65536)

	// Build the subnet info
	info := SubnetInfo{
		Index:     index,
		Network:   fmt.Sprintf("172.%d.%d.%d/%d", octet2, octet3, octet4, SubnetPrefix),
		Gateway:   fmt.Sprintf("172.%d.%d.%d", octet2, octet3, octet4+1),
		Broadcast: fmt.Sprintf("172.%d.%d.%d", octet2, octet3, octet4+15),
		UsableIPs: IPsPerSubnet - 3, // Exclude network, gateway, and broadcast
	}

	// Calculate usable range
	usableStart := fmt.Sprintf("172.%d.%d.%d", octet2, octet3, octet4+2)
	usableEnd := fmt.Sprintf("172.%d.%d.%d", octet2, octet3, octet4+14)
	info.UsableRange = fmt.Sprintf("%s-%s", usableStart, usableEnd)

	return info
}

// GetNetwork returns just the CIDR notation for a workspace ID
func GetNetwork(identifier string) string {
	info := GetSubnetInfo(identifier)
	return info.Network
}

// GetGateway returns the gateway IP for a workspace ID
func GetGateway(identifier string) string {
	info := GetSubnetInfo(identifier)
	return info.Gateway
}

// Validateidentifier checks if a workspace ID is valid (non-empty)
func ValidateIdentifier(identifier string) error {
	if identifier == "" {
		return fmt.Errorf("workspace ID cannot be empty")
	}
	return nil
}

// GetAllSubnetsInRange returns a slice of all possible subnet indices
// Useful for iteration or validation
func GetAllSubnetsInRange() []uint32 {
	subnets := make([]uint32, TotalSubnets)
	for i := range TotalSubnets {
		subnets[i] = uint32(i)
	}
	return subnets
}

// IsValidIndex checks if an index is within the valid range
func IsValidIndex(index uint32) bool {
	return index < TotalSubnets
}
