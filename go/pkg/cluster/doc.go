/*
Package cluster implements distributed cluster management with consistent hashing for Unkey's distributed architecture.

The cluster package provides a robust foundation for building distributed systems by combining node membership
management with consistent hashing. It ensures reliable workload distribution across multiple service instances
while maintaining consistency during cluster topology changes.

# Core Features

The package offers several key capabilities:

  - Automatic node discovery and health monitoring
  - Consistent hash-based workload distribution
  - Real-time cluster topology event notifications
  - Graceful node addition and removal
  - Thread-safe cluster state management

# Architecture

The cluster system is built on three main components:

 1. Membership Protocol: Uses the membership package for node discovery and failure detection
 2. Consistent Hashing: Implements a ring-based hash algorithm for workload distribution
 3. Event System: Provides real-time notifications for cluster topology changes

# Common Use Cases

This package is primarily used for:

  - Distributed rate limiting across multiple nodes
  - Workload partitioning in horizontally scaled services
  - Service discovery in microservice architectures
  - Coordinated state management across cluster nodes

# Usage

Basic cluster setup:

	cluster, err := cluster.New(cluster.Config{
		Self: cluster.Instance{
			ID:      "node-1",
			RpcAddr: "10.0.0.1:7071",
		},
		Membership: membershipService,
		Logger:     logger,
	})
	if err != nil {
		return fmt.Errorf("cluster initialization failed: %w", err)
	}
	defer cluster.Shutdown(context.Background())

Finding the responsible node for a key:

	instance, err := cluster.FindInstance(ctx, "user:123")
	if err != nil {
		return fmt.Errorf("failed to find responsible node: %w", err)
	}

Monitoring cluster changes:

	joins := cluster.SubscribeJoin()
	go func() {
		for instance := range joins {
			log.Printf("Node joined: %s at %s", instance.ID, instance.RpcAddr)
		}
	}()

# Thread Safety

All public methods in this package are thread-safe and can be called concurrently.
The internal state is protected using appropriate synchronization mechanisms.

# Error Handling

The package uses error wrapping to provide context-rich error information. Common errors include:
  - Ring initialization failures
  - Node lookup failures
  - Membership protocol errors

# Best Practices

1. Always use context.Context for operations that might need cancellation
2. Implement proper error handling for cluster operations
3. Set up monitoring for cluster health metrics
4. Use the noop implementation for testing and development

Related Packages

  - "github.com/unkeyed/unkey/go/pkg/membership": Underlying membership protocol
  - "github.com/unkeyed/unkey/go/pkg/ring": Consistent hashing implementation
  - "github.com/unkeyed/unkey/go/pkg/events": Event system for topology changes

See the Cluster interface documentation for detailed API information.
*/
package cluster
