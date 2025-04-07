/*
Package discovery implements service discovery mechanisms for distributed systems.

The package provides a unified interface for discovering peer nodes in a distributed
system, with multiple implementations suited for different deployment environments:

  - Static: Uses a predefined list of addresses. Suitable for development, testing,
    and stable production environments where node addresses are known and rarely change.

  - Redis: Implements dynamic discovery using Redis as a coordination backend.
    Nodes advertise themselves and automatically expire when offline. Best for
    containerized environments with frequent node changes.

  - AWS ECS: Discovers other tasks within the same ECS service. Suitable for
    applications running in Amazon ECS that need to form clusters.

Usage

For static discovery with known addresses:

	discoverer := discovery.Static{
		Addrs: []string{"node1:9000", "node2:9000"},
	}
	nodes, err := discoverer.Discover()

For dynamic Redis-based discovery:

	discoverer, err := discovery.NewRedis(discovery.RedisConfig{
		URL:        "redis://localhost:6379/0",
		InstanceID: "node1",
		Addr:      "10.0.1.5:9000",
		Logger:    logger,
	})
	if err != nil {
		return err
	}
	defer discoverer.Shutdown(ctx)
	
	nodes, err := discoverer.Discover()

For AWS ECS discovery:

	discoverer, err := discovery.NewAwsEcs(discovery.AwsEcsConfig{
		Region: "us-east-1",
		Logger: logger,
	})
	if err != nil {
		return err
	}
	nodes, err := discoverer.Discover()

Implementation Details

All implementations follow these principles:

  - Self-exclusion: Nodes never return their own address in discovery results
  - Error handling: Transient failures are retried where appropriate
  - Cleanup: Dynamic implementations (Redis, ECS) automatically remove offline nodes
  - Logging: Operations are logged for debugging and monitoring

The Discoverer interface is designed to be simple yet flexible enough to support
various discovery mechanisms. Custom implementations can be added by implementing
the Discoverer interface.

Thread Safety

All implementations are safe for concurrent use. The Redis implementation runs
a background goroutine for heartbeats that must be stopped via Shutdown().

Common Pitfalls

  - Always call Shutdown() on Redis discoverer to prevent resource leaks
  - Redis discovery requires nodes to share the same Redis instance
  - ECS discovery requires appropriate IAM permissions and metadata access
  - Static discovery provides no automatic node failure detection

See the individual implementation types for more detailed documentation.
*/
package discovery
