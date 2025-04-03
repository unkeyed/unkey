package cluster

import (
	"context"
)

// Instance represents an individual node in the cluster. Each Instance maintains its own
// identity and network configuration for inter-node communication.
//
// Instance objects are immutable and safe for concurrent use. They are used throughout
// the cluster package to identify and communicate with specific cluster nodes.
//
// Example:
//
//	instance := Instance{
//		ID:      "node-1",
//		RpcAddr: "10.0.0.1:7071",
//	}
type Instance struct {
	// ID uniquely identifies this instance within the cluster.
	// It must be unique across all instances and remain stable
	// for the lifetime of the instance.
	ID string

	// RpcAddr is the network address (host:port) where this instance
	// listens for RPC calls from other instances. It must be reachable
	// by all other instances in the cluster.
	RpcAddr string
}

// Cluster provides a unified interface for distributed operations across multiple instances.
// It combines membership management and consistent hashing to enable reliable distributed
// operations like rate limiting and workload distribution.
//
// All methods are safe for concurrent use. The implementation handles internal synchronization
// to maintain consistency during cluster topology changes.
//
// Example usage:
//
//	cluster, err := New(Config{
//		Self: Instance{
//			ID:      "node-1",
//			RpcAddr: "10.0.0.1:7071",
//		},
//		Membership: membershipSvc,
//		Logger:     logger,
//	})
//	if err != nil {
//		return fmt.Errorf("cluster initialization failed: %w", err)
//	}
//	defer cluster.Shutdown(context.Background())
//
//	// Find responsible instance for a key
//	instance, err := cluster.FindInstance(ctx, "user:123")
type Cluster interface {
	// Self returns information about the local instance.
	//
	// This method is thread-safe and returns an immutable Instance.
	// It's commonly used for logging, metrics, and determining if the local
	// instance is responsible for handling specific keys.
	//
	// Example:
	//
	//	self := cluster.Self()
	//	log.Printf("Local instance ID: %s", self.ID)
	Self() Instance

	// FindInstance determines which instance is responsible for a given key using
	// consistent hashing. The same key will always map to the same instance
	// (assuming the instance remains available).
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - key: Any string identifier (e.g., user ID, API key)
	//
	// Returns:
	//   - Instance: The responsible instance for the key
	//   - error: If no suitable instance is found or the cluster is unhealthy
	//
	// Errors:
	//   - ErrNoInstances: If the cluster has no available instances
	//   - ErrClusterNotInitialized: If called before proper initialization
	//   - context.Canceled: If the context is cancelled
	//
	// Example:
	//
	//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	//	defer cancel()
	//
	//	instance, err := cluster.FindInstance(ctx, "user:123")
	//	if err != nil {
	//		return fmt.Errorf("instance lookup failed: %w", err)
	//	}
	FindInstance(ctx context.Context, key string) (Instance, error)

	// Shutdown gracefully exits the cluster, notifying other instances and cleaning up resources.
	// It blocks until cleanup is complete or the context is cancelled.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//
	// Returns:
	//   - error: If shutdown fails or context expires
	//
	// Best practices:
	//   - Always call Shutdown when terminating a cluster instance
	//   - Use a timeout context to ensure shutdown doesn't hang
	//   - Handle errors to ensure proper cleanup
	//
	// Example:
	//
	//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	//	defer cancel()
	//
	//	if err := cluster.Shutdown(ctx); err != nil {
	//		log.Printf("Shutdown error: %v", err)
	//	}
	Shutdown(ctx context.Context) error

	// SubscribeJoin returns a channel that receives Instance events when new instances
	// join the cluster. The channel remains open until the cluster is shut down.
	//
	// The returned channel is receive-only and should not be closed by the caller.
	// Multiple subscribers can safely receive from their own channels concurrently.
	//
	// Best practices:
	//   - Always drain the channel in a separate goroutine
	//   - Handle slow consumers appropriately
	//   - Check for channel closure during cluster shutdown
	//
	// Example:
	//
	//	joins := cluster.SubscribeJoin()
	//	go func() {
	//		for instance := range joins {
	//			log.Printf("Instance joined: %s at %s", instance.ID, instance.RpcAddr)
	//		}
	//	}()
	SubscribeJoin() <-chan Instance

	// SubscribeLeave returns a channel that receives Instance events when instances
	// leave the cluster, either gracefully or due to failures.
	//
	// The channel behavior is identical to SubscribeJoin. Events are delivered
	// in real-time as instances leave the cluster.
	//
	// Side Effects:
	//   - May trigger rebalancing of the consistent hash ring
	//   - Affects results of FindInstance() calls
	//
	// Example:
	//
	//	leaves := cluster.SubscribeLeave()
	//	go func() {
	//		for instance := range leaves {
	//			log.Printf("Instance left: %s", instance.ID)
	//			// Handle instance removal, e.g., clean up connections
	//		}
	//	}()
	SubscribeLeave() <-chan Instance
}
