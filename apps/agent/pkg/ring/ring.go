package ring

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
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
	token string
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

	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for range t.C {
			buf := bytes.NewBuffer(nil)
			for _, token := range r.tokens {
				buf.WriteString(fmt.Sprintf("%s,", token.token))
			}

			nodes := make([]string, 0)
			for _, node := range r.nodes {
				nodes = append(nodes, node.Id)
			}
			state := sha256.Sum256(buf.Bytes())
			r.logger.Debug().Strs("nodes", nodes).Int("numTokens", len(r.tokens)).Str("state", hex.EncodeToString(state[:])).Msg("current ring state")
		}
	}()

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
		token, err := r.hash(fmt.Sprintf("%s-%d", node.Id, i))
		if err != nil {
			return err
		}
		r.tokens = append(r.tokens, Token{token: token, NodeId: node.Id})
	}
	sort.Slice(r.tokens, func(i int, j int) bool {
		return r.tokens[i].token < r.tokens[j].token
	})

	r.nodes[node.Id] = node

	r.logger.Info().Int("len", len(r.tokens)).Msg("tokens in ring")

	return nil
}

func (r *Ring[T]) RemoveNode(nodeId string) error {
	r.Lock()
	defer r.Unlock()

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

// Find returns all nodes that should own the key
// n is the number of nodes to return
// the first node in the returned slice is the primary node
// the rest are fallbacks
func (r *Ring[T]) FindNodes(key string, n int) ([]Node[T], error) {
	r.RLock()
	defer r.RUnlock()

	token, err := r.hash(key)
	if err != nil {
		return nil, err
	}
	tokenIndex := sort.Search(len(r.tokens), func(i int) bool {
		return r.tokens[i].token >= token
	})
	if tokenIndex >= len(r.tokens) {
		tokenIndex = 0
	}

	if n == 0 {
		n = 1
	}

	selectedNodes := make(map[string]Node[T])
	selected := 0
	for i := 0; selected < n && i < len(r.tokens); i++ {
		token := r.tokens[tokenIndex]
		node, ok := r.nodes[token.NodeId]
		if !ok {
			return nil, fmt.Errorf("node not found: %s", token.NodeId)
		}

		tokenIndex = (tokenIndex + 1) % len(r.tokens)
		if _, ok := selectedNodes[node.Id]; ok {
			// already selected this node
			continue
		}

		selectedNodes[node.Id] = node
		selected++
	}

	responsibleNodes := make([]Node[T], 0)
	for _, node := range selectedNodes {
		responsibleNodes = append(responsibleNodes, node)
	}
	if len(responsibleNodes) < n {
		return nil, fmt.Errorf("not enough available nodes found for key: %s", key)
	}

	return responsibleNodes, nil
}
