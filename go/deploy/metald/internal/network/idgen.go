package network

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
)

// IDGenerator generates short, unique IDs for network devices
// Network interface names in Linux are limited to 15 characters,
// so we generate 8-character IDs to leave room for prefixes like "tap-", "vh-", etc.
type IDGenerator struct {
	mu        sync.Mutex
	generated map[string]struct{} // Track generated IDs to ensure uniqueness
}

// NewIDGenerator creates a new ID generator
func NewIDGenerator() *IDGenerator {
	//exhaustruct:ignore
	return &IDGenerator{
		generated: make(map[string]struct{}),
	}
}

// GenerateNetworkID generates a unique 8-character ID for network devices
// The ID is guaranteed to be unique within this generator instance
func (g *IDGenerator) GenerateNetworkID() (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Try up to 10 times to generate a unique ID
	for range 10 {
		// Generate 4 random bytes (8 hex characters)
		bytes := make([]byte, 4)
		if _, err := rand.Read(bytes); err != nil {
			return "", fmt.Errorf("failed to generate random bytes: %w", err)
		}

		id := hex.EncodeToString(bytes)

		// Check if this ID already exists
		if _, exists := g.generated[id]; !exists {
			g.generated[id] = struct{}{}
			return id, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique ID after 10 attempts")
}

// ReleaseID removes an ID from the tracking set, allowing it to be reused
func (g *IDGenerator) ReleaseID(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.generated, id)
}

// NetworkDeviceNames holds all network device names for a VM
// AIDEV-NOTE: All names follow consistent patterns and stay within 15-char limit
type NetworkDeviceNames struct {
	ID        string // 8-character internal ID
	Namespace string // Network namespace name (no length limit)
	TAP       string // TAP device name (15 char limit)
	Bridge    string // Bridge name (15 char limit)
}

// GenerateDeviceNames creates a consistent set of network device names
func GenerateDeviceNames(networkID string) *NetworkDeviceNames {
	return &NetworkDeviceNames{
		ID:        networkID,
		Namespace: fmt.Sprintf("ns_vm_%s", networkID),
		TAP:       fmt.Sprintf("tap_%s", networkID), // 12 chars
		Bridge:    fmt.Sprintf("br-%s", networkID),
	}
}
