package discovery

// Static implements the Discoverer interface using a predefined list of addresses.
// It provides the simplest form of service discovery, suitable for:
//   - Development and testing environments
//   - Small, stable deployments with known addresses
//   - Bootstrapping more complex discovery mechanisms
//   - Fallback configuration when dynamic discovery fails
//
// Static discovery has no external dependencies and never fails, making it ideal
// for basic deployments or testing. However, it cannot detect node failures or
// handle dynamic address changes.
//
// Example usage:
//
//	discoverer := discovery.Static{
//	    Addrs: []string{
//	        "node1.example.com:9000",
//	        "node2.example.com:9000",
//	    },
//	}
//	addrs, _ := discoverer.Discover() // error is always nil
//
// To bootstrap a new cluster, use an empty address list:
//
//	discoverer := discovery.Static{} // or Addrs: []string{}
//	addrs, _ := discoverer.Discover() // returns empty slice
type Static struct {
	// Addrs contains the predefined list of node addresses to return from Discover().
	// Each address must be in "host:port" format. The host can be a DNS name or
	// IP address.
	//
	// An empty slice indicates no peers are available, signaling that a new
	// cluster should be bootstrapped.
	//
	// The slice is returned as-is without validation, so ensure addresses are
	// properly formatted when constructing the Static discoverer.
	Addrs []string
}

// Compile-time check that Static implements Discoverer
var _ Discoverer = (*Static)(nil)

// Discover returns the predefined list of addresses without modification.
// This is the simplest possible implementation of the Discoverer interface:
//   - Always returns the exact Addrs slice provided
//   - Never returns an error
//   - Thread-safe (the slice is never modified)
//   - No external dependencies or side effects
//
// The returned addresses should be in "host:port" format suitable for immediate
// use in network connections. An empty slice indicates no peers are available.
func (s Static) Discover() ([]string, error) {
	return s.Addrs, nil
}
