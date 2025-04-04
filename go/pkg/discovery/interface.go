package discovery

// Discoverer provides service discovery functionality in distributed systems.
// Implementations of this interface enable nodes to discover and connect to their
// peers dynamically, supporting various deployment environments and topologies.
//
// The interface is intentionally minimal to allow for different discovery mechanisms
// while maintaining a consistent API across the system. Common implementations include
// static address lists, Redis-based coordination, and cloud provider APIs.
//
// Thread Safety
//
// All implementations must be safe for concurrent use. Long-running implementations
// should provide a way to clean up resources (e.g., via context.Context or explicit
// shutdown methods).
//
// Implementation Requirements
//
// Implementations must:
//   - Never return their own address in the discovery results
//   - Handle transient failures appropriately (retry when possible)
//   - Clean up any resources when no longer needed
//   - Log significant operations and errors
//   - Return addresses in "host:port" format
//
// Error Handling
//
// Implementations should distinguish between:
//   - Transient errors that may resolve (network issues, temporary service unavailability)
//   - Permanent errors that require operator intervention (misconfiguration, permission issues)
//
// Common usage:
//
//	d := discovery.NewRedis(cfg)
//	addrs, err := d.Discover()
//	if err != nil {
//	    return fmt.Errorf("discovery failed: %w", err)
//	}
//	for _, addr := range addrs {
//	    // Connect to peer at addr
//	}
type Discoverer interface {
	// Discover returns a list of peer addresses available for connection.
	// Each address is returned in "host:port" format suitable for immediate use
	// in network connections.
	//
	// An empty slice with a nil error indicates no peers were found, typically
	// meaning this node should bootstrap a new cluster. This is a valid state
	// and should not be treated as an error.
	//
	// Errors should be returned for conditions that prevent discovery from
	// functioning correctly, such as network failures or invalid configuration.
	// Callers should handle errors appropriately, potentially retrying the
	// operation for transient failures.
	//
	// The returned address list explicitly excludes the current node's own
	// address to prevent self-connections.
	Discover() ([]string, error)
}
