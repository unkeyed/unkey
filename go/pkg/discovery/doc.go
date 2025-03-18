// Package discovery provides mechanisms for service discovery in distributed systems.
//
// Service discovery is a critical component in distributed architectures, allowing
// nodes to find and connect to each other without hardcoded configuration. This package
// provides a clean interface and implementations for different discovery methods:
//
// 1. Static discovery - For environments with known, stable addresses
// 2. Redis-based discovery - For dynamic environments with frequently changing nodes
//
// The package is designed to be extensible, allowing for additional discovery mechanisms
// to be implemented by conforming to the Discoverer interface.
//
// Basic usage:
//
//	// Using static discovery
//	staticDiscoverer := discovery.Static{
//	    Addrs: []string{"node1:9000", "node2:9000", "node3:9000"},
//	}
//	nodes, err := staticDiscoverer.Discover()
//
//	// Using Redis-based discovery
//	redisDiscoverer, err := discovery.NewRedis(discovery.RedisConfig{
//	    URL:    "redis://localhost:6379/0",
//	    InstanceID: "ins_123",
//	    Addr:   "10.0.1.5:9000",
//	    Logger: logger,
//	})
//	if err != nil {
//	    return err
//	}
//	defer redisDiscoverer.Shutdown(ctx)
//
//	nodes, err := redisDiscoverer.Discover()
//
// The discovery package is typically used in conjunction with membership systems
// to form and maintain clusters of cooperating nodes.
package discovery
