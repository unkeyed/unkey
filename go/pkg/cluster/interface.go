package cluster

import (
	"context"
)

type Node struct {
	ID      string
	RpcAddr string
	Addr    string
}

// Cluster abstracts away membership and consistent hashing.
type Cluster interface {
	Self() Node

	FindNode(ctx context.Context, key string) (Node, error)
	Shutdown(ctx context.Context) error

	SubscribeJoin() <-chan Node
	SubscribeLeave() <-chan Node
}
