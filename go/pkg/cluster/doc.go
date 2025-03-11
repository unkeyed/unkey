// Package cluster provides abstractions for distributed cluster membership,
// consistent hashing, and node management in a multi-node environment.
//
// It combines membership protocols with consistent hashing to enable
// reliable distributed operations, such as distributed rate limiting,
// across multiple service instances. The package handles node discovery,
// failure detection, and request routing to the appropriate node.
//
// Key features:
// - Node discovery and membership tracking
// - Consistent hashing for stable workload distribution
// - Event subscriptions for node join/leave events
// - Automatic handling of cluster topology changes
//
// The cluster package is built on top of the membership package which
// provides the underlying node discovery and failure detection mechanisms.
// It adds a consistent hash ring to distribute workloads evenly across nodes
// while minimizing redistribution when the cluster topology changes.
//
// Example usage:
//
//	// Create a cluster instance
//	cluster, err := cluster.New(cluster.Config{
//	    Self: cluster.Node{
//	        ID:      "node-1",
//	        Addr:    "10.0.0.1",
//	        RpcAddr: "10.0.0.1:7071",
//	    },
//	    Membership: membershipService,
//	    Logger:     logger,
//	    RpcPort:    7071,
//	})
//	if err != nil {
//	    return fmt.Errorf("failed to create cluster: %w", err)
//	}
//
//	// Find the responsible node for a given key
//	responsibleNode, err := cluster.FindNode(ctx, "user:123")
//	if err != nil {
//	    return fmt.Errorf("failed to find node: %w", err)
//	}
//
//	// Listen for node join events
//	joinCh := cluster.SubscribeJoin()
//	go func() {
//	    for node := range joinCh {
//	        log.Printf("Node joined: %s at %s", node.ID, node.Addr)
//	    }
//	}()
//
//	// Graceful shutdown
//	defer cluster.Shutdown(ctx)
package cluster
