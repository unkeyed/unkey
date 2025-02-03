package cluster

import "context"

type Node struct {
	ID      string
	RpcAddr string
}

// Cluster abstracts away membership and consistent hashing.
type Cluster interface {
	FindNode(ctx context.Context, key string) (Node, error)
	ShutDown(ctx context.Context) error
}
