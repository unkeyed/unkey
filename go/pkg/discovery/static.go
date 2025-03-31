package discovery

// Static implements the Discoverer interface using a pre-defined list of addresses.
// This discovery method is suitable for environments where node addresses are
// known in advance and don't change frequently, such as development environments
// or small, stable deployments.
type Static struct {
	// Addrs contains the list of node addresses to return from Discover().
	// Each address should be in a format that allows direct communication with the node,
	// typically "host:port".
	Addrs []string
}

var _ Discoverer = (*Static)(nil)

// Discover returns the predefined list of node addresses.
// This implementation simply returns the Addrs slice without any additional logic.
// It never returns an error.
//
// For test environments or bootstrapping scenarios, an empty slice can be used
// to indicate that a new cluster should be formed rather than joining an existing one.
func (s Static) Discover() ([]string, error) {
	return s.Addrs, nil
}
