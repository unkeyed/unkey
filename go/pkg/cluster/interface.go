package cluster

import (
	"context"
	"net"
)

type Node struct {
	ID      string
	RpcAddr string
	Addr    net.IP
}

// Cluster abstracts away membership and consistent hashing.
type Cluster interface {
	Self() Node

	FindNode(ctx context.Context, key string) (Node, error)
	Shutdown(ctx context.Context) error

	SubscribeJoin() <-chan Node
	SubscribeLeave() <-chan Node
}
