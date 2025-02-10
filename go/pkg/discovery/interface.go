package discovery

type Discoverer interface {
	// Discover returns an slice of addresses to join.
	//
	// If an empty slice is returned, you should bootstrap the cluster.
	Discover() ([]string, error)
}
