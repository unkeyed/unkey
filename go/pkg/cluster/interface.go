package cluster

import (
	"context"
)

// Instance represents an individual instance in the cluster.
// It contains identifying information and network addresses
// needed for instance-to-instance communication.
type Instance struct {
	// ID is the unique identifier for this instance within the cluster
	ID string

	// RpcAddr is the address (host:port) where the instance listens for RPC calls
	RpcAddr string
}

// Cluster abstracts away membership and consistent hashing to provide
// a unified interface for distributed operations across multiple instances.
//
// The cluster interface handles:
// - Finding the responsible instance for a given key
// - Tracking instances joining and leaving the cluster
// - Providing information about the local instance
// - Managing graceful cluster exit
type Cluster interface {
	// Self returns information about the local instance.
	// This is useful for identifying the current instance in logs and metrics.
	Self() Instance

	// FindInstance determines which instance is responsible for a given key
	// based on consistent hashing. This ensures that the same key
	// is always routed to the same instance (as long as the instance is available).
	//
	// The key can be any string identifier, such as a user ID, API key,
	// or resource name. The method returns the instance that should handle
	// operations for this key.
	//
	// Returns an error if no suitable instance is found or if the cluster
	// is not properly initialized.
	FindInstance(ctx context.Context, key string) (Instance, error)

	// Shutdown gracefully exits the cluster, notifying other instances
	// and cleaning up resources.
	//
	// This should be called when the instance is being taken down to ensure
	// proper cluster rebalancing and to avoid false failure detection.
	Shutdown(ctx context.Context) error

	// SubscribeJoin returns a channel that receives Instance events
	// whenever a new instance joins the cluster.
	//
	// This can be used to react to cluster topology changes, such as
	// pre-warming caches or rebalancing workloads.
	SubscribeJoin() <-chan Instance

	// SubscribeLeave returns a channel that receives Instance events
	// whenever a instance leaves the cluster.
	//
	// This can be used to handle graceful degradation or workload
	// redistribution when instances become unavailable.
	SubscribeLeave() <-chan Instance
}
