package main

import (
	"fmt"
	"hash/fnv"
	"math"
)

// Original function with issues
func CalculateNetworkMapIDOriginal(workspaceID string) uint32 {
	hash := fnv.New32a()
	hash.Write([]byte(workspaceID))
	return uint32(int(hash.Sum32()) % int(math.Pow(2, (28-12))))
}

// Improved version - returns 0 to 65535
func CalculateNetworkMapID(workspaceID string) uint32 {
	hash := fnv.New32a()
	hash.Write([]byte(workspaceID))
	return hash.Sum32() % 65536 // or % (1 << 16)
}

// If you need 1-65536 instead of 0-65535
func CalculateNetworkMapIDOneBased(workspaceID string) uint32 {
	hash := fnv.New32a()
	hash.Write([]byte(workspaceID))
	return (hash.Sum32() % 65536) + 1
}

// Even more efficient using bitwise operations
func CalculateNetworkMapIDFast(workspaceID string) uint32 {
	hash := fnv.New32a()
	hash.Write([]byte(workspaceID))
	return hash.Sum32() & 0xFFFF // Masks to keep only lower 16 bits (0-65535)
}

// If you want to directly map to a subnet index (0-based)
func GetSubnetIndex(workspaceID string) uint32 {
	hash := fnv.New32a()
	hash.Write([]byte(workspaceID))
	return hash.Sum32() & 0xFFFF // 0-65535
}

// Get the actual subnet CIDR for a workspace
func GetSubnetForWorkspace(workspaceID string) string {
	index := GetSubnetIndex(workspaceID)

	// Each /28 subnet has 16 IPs
	// Calculate which subnet this index maps to
	subnetNumber := index * 16

	// Calculate octets for 172.16.0.0/12 base
	// We have 20 bits to work with (32-12)
	// The first 4 bits determine the second octet (16-31)
	// The next 8 bits determine the third octet (0-255)
	// The last 8 bits determine the fourth octet (0-255)

	totalOffset := subnetNumber
	octet4 := totalOffset % 256
	octet3 := (totalOffset / 256) % 256
	octet2 := 16 + (totalOffset / 65536)

	return fmt.Sprintf("172.%d.%d.%d/28", octet2, octet3, octet4)
}

func main() {
	testIDs := []string{
		"workspace-123",
		"workspace-456",
		"workspace-789",
		"test",
		"production",
	}

	fmt.Println("Workspace ID -> Subnet Mapping")
	fmt.Println("--------------------------------")

	for _, id := range testIDs {
		index := GetSubnetIndex(id)
		subnet := GetSubnetForWorkspace(id)
		fmt.Printf("%-15s -> Index: %5d, Subnet: %s\n", id, index, subnet)
	}

	// Test distribution
	fmt.Println("\n--- Distribution Test ---")
	testDistribution()
}

// Test the distribution quality of the hash function
func testDistribution() {
	buckets := make(map[uint32]int)
	numTests := 100000

	for i := 0; i < numTests; i++ {
		workspaceID := fmt.Sprintf("workspace-%d", i)
		index := GetSubnetIndex(workspaceID)
		bucketIndex := index / 6554 // Divide into 10 buckets
		buckets[bucketIndex]++
	}

	fmt.Printf("Distribution across %d samples:\n", numTests)
	for i := uint32(0); i < 10; i++ {
		percentage := float64(buckets[i]) / float64(numTests) * 100
		fmt.Printf("Bucket %d: %.2f%%\n", i, percentage)
	}
}
