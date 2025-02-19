package ring

import (
	"context"
	// We use it as fast and insecure hash
	// nolint:gosec
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"log/slog"
	"sort"
	"sync"

	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/logging"
)

// Node represents an individual entity in the ring, usually a container instance.
// Nodes are identified by their unique ID and can have arbitrary tags associated with them.
// Tags must be copyable, don't use pointers or channels.
type Node[T any] struct {
	// The id must be unique across all nodes in the ring and ideally should be stable
	// across restarts of the node to minimize data movement.
	ID string
	// Arbitrary tags associated with the node
	// For example an ip address, availability zone, etc.
	// Nodes may get copied or cached, so don't use pointers or channels in tags
	Tags T
}
type Config struct {
	// how many tokens each node should have
	TokensPerNode int

	Logger logging.Logger
}

type token struct {
	hash   uint64
	nodeID string
}

type Ring[T any] struct {
	mu sync.RWMutex

	tokensPerNode int
	// nodeIDs
	nodes  map[string]Node[T]
	tokens []token
	logger logging.Logger
}

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

func (r *Ring[T]) AddNode(ctx context.Context, node Node[T]) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, n := range r.nodes {
		if n.ID == node.ID {
			return fmt.Errorf("node already exists: %s", node.ID)
		}
	}
	r.logger.Info(ctx, "adding node to ring", slog.String("newNodeID", node.ID))

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

	r.logger.Info(ctx, "tokens in ring", slog.Int("nodes", len(r.nodes)), slog.Int("tokens", len(r.tokens)))

	return nil
}

func (r *Ring[T]) RemoveNode(ctx context.Context, nodeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logger.Info(ctx, "removing node from ring", slog.String("removedNodeID", nodeID))

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

func (r *Ring[T]) hash(key string) (uint64, error) {
	// We use it as fast and insecure hash
	// nolint:gosec
	sum := md5.Sum([]byte(key))

	return binary.BigEndian.Uint64(sum[:8]), nil
}

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
