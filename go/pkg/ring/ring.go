package ring

import (
	"context"
	// We use it as fast and insecure hash
	// nolint:gosec
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"sort"
	"sync"

	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Node represents an individual entity in the ring, usually a service instance
// or server. Nodes are identified by a unique ID and can carry arbitrary
// type-safe data in the Tags field.
//
// The generic parameter T allows storing any data type with the node, such as
// network addresses, capabilities, or metadata.
//
// Example:
//
//	// A node with string data
//	strNode := ring.Node[string]{
//	    ID: "node1",
//	    Tags: "http://10.0.0.1:8080",
//	}
//
//	// A node with structured data
//	type ServerInfo struct {
//	    Addr    string
//	    Region  string
//	    Healthy bool
//	}
//
//	serverNode := ring.Node[ServerInfo]{
//	    ID: "node2",
//	    Tags: ServerInfo{
//	        Addr:    "10.0.0.2:8080",
//	        Region:  "us-west-1",
//	        Healthy: true,
//	    },
//	}
type Node[T any] struct {
	// ID uniquely identifies this node within the ring.
	// IDs should be stable across restarts to minimize data redistribution.
	ID string

	// Tags stores arbitrary data associated with this node.
	// The data should be copyable, so avoid using pointers, channels,
	// or other non-copyable types.
	Tags T
}

// Config defines the parameters for creating a new consistent hash ring.
type Config struct {
	// TokensPerNode specifies how many virtual nodes (tokens) each physical
	// node will have on the ring. More tokens lead to better distribution
	// but consume more memory.
	//
	// Recommended values are between 64 and 256 for most applications.
	TokensPerNode int

	// Logger is used to record significant ring operations like node additions
	// and removals. If nil, logging is disabled.
	Logger logging.Logger
}

// token represents a point on the consistent hash ring.
// Each physical node is represented by multiple tokens.
type token struct {
	// hash is the position of this token on the ring
	hash uint64
	// nodeID identifies which physical node owns this token
	nodeID string
}

// Ring implements a consistent hash ring with support for virtual nodes.
// It is parameterized by the node data type T and is safe for concurrent use.
//
// Ring operations are thread-safe, allowing simultaneous node lookups
// while the ring membership is being modified.
type Ring[T any] struct {
	mu sync.RWMutex

	tokensPerNode int
	// nodeIDs
	nodes  map[string]Node[T]
	tokens []token
	logger logging.Logger
}

// New creates a new consistent hash ring with the specified configuration.
// The generic parameter T defines the type of data associated with each node.
//
// The returned ring is initially empty and nodes must be added via AddNode.
//
// Example:
//
//	// Create a ring with 128 tokens per node
//	r, err := ring.New[ServerInfo](ring.Config{
//	    TokensPerNode: 128,
//	    Logger: logger,
//	})
//	if err != nil {
//	    log.Fatalf("Failed to create ring: %v", err)
//	}
func New[T any](config Config) (*Ring[T], error) {
	r := &Ring[T]{
		mu:            sync.RWMutex{},
		tokensPerNode: config.TokensPerNode,
		logger:        config.Logger,
		nodes:         make(map[string]Node[T]),
		tokens:        make([]token, 0),
	}

	return r, nil
}

// AddNode registers a new node in the consistent hash ring.
// It creates virtual nodes (tokens) for the physical node and distributes
// them evenly around the ring.
//
// The node ID must be unique. If a node with the same ID already exists,
// an error is returned.
//
// This operation has O(n log n) time complexity where n is the number of
// total tokens in the ring after addition.
//
// Example:
//
//	err := ring.AddNode(ctx, ring.Node[ServerInfo]{
//	    ID: "server1",
//	    Tags: ServerInfo{
//	        Addr: "10.0.0.1",
//	        Port: 8080,
//	    },
//	})
//	if err != nil {
//	    log.Printf("Failed to add node: %v", err)
//	}
func (r *Ring[T]) AddNode(ctx context.Context, node Node[T]) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, n := range r.nodes {
		if n.ID == node.ID {
			return fmt.Errorf("node already exists: %s", node.ID)
		}
	}
	r.logger.Info("adding node to ring", "newNodeID", node.ID)

	for i := 0; i < r.tokensPerNode; i++ {
		hash, err := r.hash(fmt.Sprintf("%s-%d", node.ID, i))
		if err != nil {
			return err
		}
		r.tokens = append(r.tokens, token{hash: hash, nodeID: node.ID})
	}
	sort.Slice(r.tokens, func(i int, j int) bool {
		return r.tokens[i].hash < r.tokens[j].hash
	})

	r.nodes[node.ID] = node

	r.logger.Info("tokens in ring",
		"nodes", len(r.nodes),
		"tokens", len(r.tokens),
	)

	return nil
}

// RemoveNode removes a node and all its tokens from the ring.
// If the node does not exist, the operation is a no-op and no error is returned.
//
// After removal, keys that were previously assigned to this node will be
// redistributed to other nodes in the ring based on the consistent hashing algorithm.
//
// This operation has O(n) time complexity where n is the total number of
// tokens in the ring.
//
// Example:
//
//	err := ring.RemoveNode(ctx, "server1")
//	if err != nil {
//	    log.Printf("Failed to remove node: %v", err)
//	}
func (r *Ring[T]) RemoveNode(ctx context.Context, nodeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logger.Info("removing node from ring", "removedNodeID", nodeID)

	delete(r.nodes, nodeID)

	tokens := make([]token, 0)
	for _, t := range r.tokens {
		if t.nodeID != nodeID {
			tokens = append(tokens, t)
		}
	}
	r.tokens = tokens

	return nil
}

// hash computes the hash of a key as a uint64.
// It uses MD5 for its uniform distribution properties.
// The first 8 bytes of the MD5 hash are converted to a uint64.
func (r *Ring[T]) hash(key string) (uint64, error) {
	// We use it as fast and insecure hash
	// nolint:gosec
	sum := md5.Sum([]byte(key))

	return binary.BigEndian.Uint64(sum[:8]), nil
}

// Members returns a list of all nodes currently in the ring.
// The returned slice is a copy of the internal nodes map, so it can be
// safely modified by the caller.
//
// Example:
//
//	nodes := ring.Members()
//	for _, node := range nodes {
//	    fmt.Printf("Node %s at %s\n", node.ID, node.Tags.Addr)
//	}
func (r *Ring[T]) Members() []Node[T] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	nodes := make([]Node[T], len(r.nodes))
	i := 0
	for _, n := range r.nodes {
		nodes[i] = n
		i++
	}
	return nodes
}

// FindNode determines which node is responsible for the given key.
// It hashes the key to locate it on the ring, then finds the next node
// in a clockwise direction.
//
// If the ring is empty, an error is returned.
//
// This operation has O(log n) time complexity where n is the total number
// of tokens in the ring.
//
// Example:
//
//	// Find node responsible for a user
//	node, err := ring.FindNode("user:12345")
//	if err != nil {
//	    log.Printf("Failed to find node: %v", err)
//	    return
//	}
//
//	// Use the node information
//	client := NewClient(node.Tags.Addr)
func (r *Ring[T]) FindNode(key string) (Node[T], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.tokens) == 0 {
		return Node[T]{}, fault.New("ring is empty")

	}

	hash, err := r.hash(key)
	if err != nil {
		return Node[T]{}, err
	}
	tokenIndex := sort.Search(len(r.tokens), func(i int) bool {
		return r.tokens[i].hash >= hash
	})
	if tokenIndex >= len(r.tokens) {
		tokenIndex = 0
	}

	token := r.tokens[tokenIndex]
	node, ok := r.nodes[token.nodeID]
	if !ok {
		return Node[T]{}, fmt.Errorf("node not found: %s", token.nodeID)

	}

	return node, nil
}
