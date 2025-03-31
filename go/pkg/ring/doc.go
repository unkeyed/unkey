// Package ring implements a consistent hashing ring for distributed systems.
//
// Consistent hashing provides a way to distribute data or workload across multiple
// nodes while minimizing redistribution when the set of nodes changes. This is
// particularly useful in distributed caches, databases, and service discovery systems.
//
// The implementation uses a virtual node approach where each physical node is
// represented by multiple points (tokens) on the hash ring. This technique helps
// achieve a more uniform distribution even with a small number of physical nodes.
//
// Key features:
//   - Generic implementation that works with any node data type
//   - Configurable number of virtual nodes per physical node
//   - Thread-safe operations for concurrent use
//   - Support for node addition and removal with minimal redistribution
//   - Deterministic node selection for any given key
//
// Basic usage:
//
//	// Define a type for node data
//	type ServerNode struct {
//	    Addr string
//	    Port int
//	}
//
//	// Create a new ring with 128 virtual nodes per physical node
//	r, err := ring.New[ServerNode](ring.Config{
//	    TokensPerNode: 128,
//	    Logger: logger,
//	})
//	if err != nil {
//	    log.Fatalf("Failed to create ring: %v", err)
//	}
//
//	// Add nodes to the ring
//	err = r.AddNode(ctx, ring.Node[ServerNode]{
//	    ID: "node1",
//	    Tags: ServerNode{Addr: "10.0.0.1", Port: 8080},
//	})
//
//	// Find the node responsible for a key
//	node, err := r.FindNode("user:12345")
//	if err != nil {
//	    log.Printf("Failed to find node: %v", err)
//	    return
//	}
//
//	// Use the node information
//	conn := connect(node.Tags.Addr, node.Tags.Port)
//
// The ring uses MD5 hashing to map both nodes and keys to points on the ring.
// While not cryptographically secure, MD5 provides sufficient distribution
// properties for consistent hashing.
package ring
