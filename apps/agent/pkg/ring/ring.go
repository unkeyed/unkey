package ring

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"sort"
	"sync"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
)

type Node struct {
	Id      string
	RpcAddr string
}
type Config struct {
	// how many tokens each node should have
	TokensPerNode int

	Logger logging.Logger
}

type Token struct {
	token uint64
	// index into the nodeIds array
	NodeId string
}

type Ring struct {
	sync.RWMutex

	tokensPerNode int
	// nodeIds
	nodes  map[string]*Node
	tokens []Token
	logger logging.Logger
}

func New(config Config) (*Ring, error) {
	r := &Ring{
		tokensPerNode: config.TokensPerNode,
		logger:        config.Logger,
		nodes:         make(map[string]*Node),
		tokens:        make([]Token, 0),
	}

	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for range t.C {
			buf := bytes.NewBuffer(nil)
			for _, token := range r.tokens {
				buf.WriteString(fmt.Sprintf("%d,", token.token))
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

func (r *Ring) AddNode(node Node) error {
	r.Lock()
	defer r.Unlock()

	r.logger.Info().Str("nodeId", node.Id).Msg("adding node to ring")
	for _, n := range r.nodes {
		if n.Id == node.Id {
			return fmt.Errorf("node already exists: %s", node.Id)
		}

	}

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

	r.nodes[node.Id] = &node

	return nil
}

func (r *Ring) RemoveNode(nodeId string) error {
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

func (r *Ring) hash(key string) (uint64, error) {

	h := fnv.New64a()
	_, err := h.Write([]byte(key))
	if err != nil {
		return 0, err
	}
	return h.Sum64(), nil
}

func (r *Ring) Members() []Node {
	r.RLock()
	defer r.RUnlock()

	nodes := make([]Node, len(r.nodes))
	i := 0
	for _, n := range r.nodes {
		nodes[i] = *n
		i++
	}
	return nodes
}

// Find returns the node that owns the key
func (r *Ring) FindNode(key string) (*Node, error) {
	r.RLock()
	defer r.RUnlock()

	t := []uint64{}
	for _, token := range r.tokens {
		t = append(t, token.token)
	}

	hash, err := r.hash(key)
	if err != nil {
		return nil, fmt.Errorf("failed to hash key: %s", key)
	}

	tokenIndex := sort.Search(len(r.tokens), func(i int) bool {
		return r.tokens[i].token >= hash
	})
	if tokenIndex >= len(r.tokens) {
		tokenIndex = 0
	}

	token := r.tokens[tokenIndex]
	node, ok := r.nodes[token.NodeId]
	if !ok {
		return nil, fmt.Errorf("node not found: %s", token.NodeId)
	}

	return node, nil
}
