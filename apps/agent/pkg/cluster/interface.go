package cluster

type Cluster interface {
	Shutdown() error
	FindNode(key string) (Node, error)
	AuthToken() string

	// Returns its own node ID
	NodeId() string

	// Returns the number of nodes in the cluster
	Size() int
}
