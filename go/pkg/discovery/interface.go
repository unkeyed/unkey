package discovery

// Discoverer defines the interface for service discovery mechanisms.
// Implementations of this interface provide a way to discover other nodes
// in a distributed system, enabling dynamic cluster formation.
type Discoverer interface {
	// Discover returns a slice of addresses to join.
	// Each address should be in a format that can be used to establish
	// direct communication with the node, typically "host:port".
	//
	// If an empty slice is returned, it indicates there are no existing nodes
	// to join, and the caller should bootstrap a new cluster.
	//
	// If an error occurs during discovery, it should be returned to allow
	// the caller to handle the failure appropriately.
	Discover() ([]string, error)
}
