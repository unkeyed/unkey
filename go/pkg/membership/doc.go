// Package membership implements cluster membership and node discovery.
//
// It provides mechanisms for nodes to discover and communicate with each
// other in a distributed environment. The package supports multiple discovery
// methods including static addresses and Redis-based discovery.
//
// Membership uses a gossip protocol for efficient peer-to-peer communication
// and failure detection, ensuring consistent cluster state across all nodes.
package membership
