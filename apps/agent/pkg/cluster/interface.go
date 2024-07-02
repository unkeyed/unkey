package cluster

type Cluster interface {
	Join(addrs []string) (clusterSize int, err error)
	Shutdown() error
	FindNodes(key string, n int) ([]Node, error)
	FindNode(key string) (Node, error)
	AuthToken() string

	// Returns its own node ID
	NodeId() string
}
