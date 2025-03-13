package cluster

import (
	"context"
)

// noop provides a no-operation implementation of the Cluster interface.
// It's useful for testing or for deployments that don't require clustering.
type noop struct {
	self Node
}

// Ensure noop implements the Cluster interface
var _ Cluster = (*noop)(nil)

// NewNoop creates a new no-operation cluster implementation.
// It returns a Cluster that doesn't perform any actual clustering
// operations but satisfies the interface.
//
// This is useful for:
// - Single-node deployments where clustering is unnecessary
// - Development environments
// - Testing scenarios where real clustering would be overkill
//
// Example:
//
//	// Create a no-op cluster for local development
//	cluster := cluster.NewNoop("local-node", "localhost")
func NewNoop(id string, host string) *noop {
	return &noop{self: Node{
		ID:      id,
		RpcAddr: "",
	}}
}

// Self returns information about the local node.
func (n *noop) Self() Node {
	return n.self
}

// FindNode always returns the local node, since there's only
// one node in a no-op cluster.
func (n *noop) FindNode(ctx context.Context, key string) (Node, error) {
	return n.self, nil

}

// Shutdown is a no-op that always succeeds.
func (n *noop) Shutdown(ctx context.Context) error {
	return nil
}

// SubscribeJoin returns a never-closing, never-sending channel.
func (n *noop) SubscribeJoin() <-chan Node {
	return make(chan Node)
}

// SubscribeLeave returns a never-closing, never-sending channel.
func (n *noop) SubscribeLeave() <-chan Node {
	return make(chan Node)
}
