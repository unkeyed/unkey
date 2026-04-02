package balancer

// Balancer selects an instance from a set of candidates.
// Implementations must be safe for concurrent use.
type Balancer interface {
	// Pick selects an instance from instanceIDs and returns its index.
	// The slice is guaranteed to be non-empty.
	Pick(instanceIDs []string) int
}

// InflightTracker optionally tracks in-flight requests per instance.
// Balancers that need load awareness (like P2C) implement this in addition
// to Balancer. Stateless balancers (random, round-robin) only need Balancer.
type InflightTracker interface {
	Acquire(instanceID string)
	Release(instanceID string)
}
