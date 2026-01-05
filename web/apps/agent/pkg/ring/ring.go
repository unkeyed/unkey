package ring

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/Southclaws/fault"
	"github.com/unkeyed/unkey/svc/agent/pkg/logging"
	"github.com/unkeyed/unkey/svc/agent/pkg/repeat"
)

// Node represents an individual entity in the ring, usually a container instance.
// Nodes are identified by their unique ID and can have arbitrary tags associated with them.
// Tags must be copyable, don't use pointers or channels.
type Node[T any] struct {
	// The id must be unique across all nodes in the ring and ideally should be stable
	// across restarts of the node to minimize data movement.
	Id string
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

type Token struct {
	hash string
	// index into the nodeIds array
	NodeId string
}

type Ring[T any] struct {
	sync.RWMutex

	tokensPerNode int
	// nodeIds
	nodes  map[string]Node[T]
	tokens []Token
	logger logging.Logger
}

func New[T any](config Config) (*Ring[T], error) {
	r := &Ring[T]{
		tokensPerNode: config.TokensPerNode,
		logger:        config.Logger,
		nodes:         make(map[string]Node[T]),
		tokens:        make([]Token, 0),
	}

	repeat.Every(10*time.Second, func() {
		r.Lock()
		defer r.Unlock()
		buf := bytes.NewBuffer(nil)
		for _, token := range r.tokens {
			_, err := buf.WriteString(fmt.Sprintf("%s,", token.hash))
			if err != nil {
				r.logger.Error().Err(err).Msg("failed to write token to buffer")
			}
			continue
		}

		ringTokens.Set(float64(len(r.tokens)))

	})

	return r, nil
}

func (r *Ring[T]) AddNode(node Node[T]) error {
	r.Lock()
	defer r.Unlock()

	for _, n := range r.nodes {
		if n.Id == node.Id {
			return fmt.Errorf("node already exists: %s", node.Id)
		}
	}
	r.logger.Info().Str("newNodeId", node.Id).Msg("adding node to ring")

	for i := 0; i < r.tokensPerNode; i++ {
		hash, err := r.hash(fmt.Sprintf("%s-%d", node.Id, i))
		if err != nil {
			return err
		}
		r.tokens = append(r.tokens, Token{hash: hash, NodeId: node.Id})
	}
	sort.Slice(r.tokens, func(i int, j int) bool {
		return r.tokens[i].hash < r.tokens[j].hash
	})

	r.nodes[node.Id] = node

	r.logger.Info().Int("nodes", len(r.nodes)).Int("tokens", len(r.tokens)).Msg("tokens in ring")

	return nil
}

func (r *Ring[T]) RemoveNode(nodeId string) error {
	r.Lock()
	defer r.Unlock()
	r.logger.Info().Str("removedNodeId", nodeId).Msg("removing node from ring")

	delete(r.nodes, nodeId)

	tokens := make([]Token, 0)
	for _, t := range r.tokens {
		if t.NodeId != nodeId {
			tokens = append(tokens, t)
		}
	}
	r.tokens = tokens

	return nil
}

func (r *Ring[T]) hash(key string) (string, error) {

	h := sha256.New()
	_, err := h.Write([]byte(key))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

func (r *Ring[T]) Members() []Node[T] {
	r.RLock()
	defer r.RUnlock()

	nodes := make([]Node[T], len(r.nodes))
	i := 0
	for _, n := range r.nodes {
		nodes[i] = n
		i++
	}
	return nodes
}

func (r *Ring[T]) FindNode(key string) (Node[T], error) {
	r.RLock()
	defer r.RUnlock()

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
	node, ok := r.nodes[token.NodeId]
	if !ok {
		return Node[T]{}, fmt.Errorf("node not found: %s", token.NodeId)

	}

	foundNode.WithLabelValues(key, node.Id).Inc()

	return node, nil
}
