package cluster

import (
	"context"
)

// Node represents an individual node in the cluster.
// It contains identifying information and network addresses
// needed for node-to-node communication.
type Node struct {
	// ID is the unique identifier for this node within the cluster
	ID string

	// RpcAddr is the address (host:port) where the node listens for RPC calls
	RpcAddr string
}

// Cluster abstracts away membership and consistent hashing to provide
// a unified interface for distributed operations across multiple nodes.
//
// The cluster interface handles:
// - Finding the responsible node for a given key
// - Tracking nodes joining and leaving the cluster
// - Providing information about the local node
// - Managing graceful cluster exit
type Cluster interface {
	// Self returns information about the local node.
	// This is useful for identifying the current node in logs and metrics.
	Self() Node

	// FindNode determines which node is responsible for a given key
	// based on consistent hashing. This ensures that the same key
	// is always routed to the same node (as long as the node is available).
	//
	// The key can be any string identifier, such as a user ID, API key,
	// or resource name. The method returns the node that should handle
	// operations for this key.
	//
	// Returns an error if no suitable node is found or if the cluster
	// is not properly initialized.
	FindNode(ctx context.Context, key string) (Node, error)

	// Shutdown gracefully exits the cluster, notifying other nodes
	// and cleaning up resources.
	//
	// This should be called when the node is being taken down to ensure
	// proper cluster rebalancing and to avoid false failure detection.
	Shutdown(ctx context.Context) error

	// SubscribeJoin returns a channel that receives Node events
	// whenever a new node joins the cluster.
	//
	// This can be used to react to cluster topology changes, such as
	// pre-warming caches or rebalancing workloads.
	SubscribeJoin() <-chan Node

	// SubscribeLeave returns a channel that receives Node events
	// whenever a node leaves the cluster.
	//
	// This can be used to handle graceful degradation or workload
	// redistribution when nodes become unavailable.
	SubscribeLeave() <-chan Node
}
