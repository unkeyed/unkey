package cluster

import (
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/memberlist"
	clusterv1 "github.com/unkeyed/unkey/gen/proto/cluster/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"google.golang.org/protobuf/proto"
)

const maxJoinAttempts = 10

// Cluster is the public interface for gossip-based cluster membership.
type Cluster interface {
	Broadcast(msg clusterv1.IsClusterMessage_Payload) error
	Members() []*memberlist.Node
	IsGateway() bool
	WANAddr() string
	Close() error
}

// gossipCluster manages a two-tier gossip membership: a LAN pool for intra-region
// communication and, on the elected gateway node, a WAN pool for cross-region
// communication.
type gossipCluster struct {
	config Config

	mu        sync.RWMutex
	lan       *memberlist.Memberlist
	lanQueue  *memberlist.TransmitLimitedQueue
	wan       *memberlist.Memberlist
	wanQueue  *memberlist.TransmitLimitedQueue
	isGateway bool
	closing   atomic.Bool

	// evalCh is used to trigger async gateway evaluation from memberlist
	// callbacks. This avoids calling Members() inside NotifyJoin/NotifyLeave
	// where memberlist holds its internal state lock.
	evalCh chan struct{}
	done   chan struct{}
}

// New creates a new cluster node, starts the LAN memberlist, joins LAN seeds,
// and begins gateway evaluation.
func New(cfg Config) (Cluster, error) {
	cfg.setDefaults()

	c := &gossipCluster{
		config:    cfg,
		mu:        sync.RWMutex{},
		lan:       nil,
		lanQueue:  nil,
		wan:       nil,
		wanQueue:  nil,
		isGateway: false,
		closing:   atomic.Bool{},
		evalCh:    make(chan struct{}, 1),
		done:      make(chan struct{}),
	}

	// Start the async gateway evaluator
	go c.gatewayEvalLoop()

	// Configure LAN memberlist
	lanCfg := memberlist.DefaultLANConfig()
	lanCfg.Name = cfg.NodeID
	lanCfg.BindAddr = cfg.BindAddr
	lanCfg.BindPort = cfg.BindPort
	lanCfg.AdvertisePort = cfg.BindPort
	lanCfg.LogOutput = io.Discard
	lanCfg.Delegate = newLANDelegate(c)
	lanCfg.Events = newLANEventDelegate(c)

	lan, err := memberlist.Create(lanCfg)
	if err != nil {
		close(c.done)
		return nil, fmt.Errorf("failed to create LAN memberlist: %w", err)
	}

	c.mu.Lock()
	c.lan = lan
	c.lanQueue = &memberlist.TransmitLimitedQueue{
		NumNodes:       func() int { return lan.NumMembers() },
		RetransmitMult: 3,
	}
	c.mu.Unlock()

	// Join LAN seeds with retries — the headless service DNS may not be
	// resolvable immediately at pod startup.
	if len(cfg.LANSeeds) > 0 {
		go c.joinSeeds("LAN", func() *memberlist.Memberlist {
			c.mu.RLock()
			defer c.mu.RUnlock()
			return c.lan
		}, cfg.LANSeeds, c.triggerEvalGateway)
	}

	// Trigger initial gateway evaluation
	c.triggerEvalGateway()

	return c, nil
}

// joinSeeds attempts to join seeds on the given memberlist with exponential backoff.
// pool is used for logging ("LAN" or "WAN"). onSuccess is called after a successful join.
func (c *gossipCluster) joinSeeds(pool string, list func() *memberlist.Memberlist, seeds []string, onSuccess func()) {
	backoff := 500 * time.Millisecond

	for attempt := 1; attempt <= maxJoinAttempts; attempt++ {
		select {
		case <-c.done:
			return
		default:
		}

		ml := list()
		if ml == nil {
			return
		}

		_, err := ml.Join(seeds)
		if err == nil {
			logger.Info("Joined "+pool+" seeds", "seeds", seeds, "attempt", attempt)
			if onSuccess != nil {
				onSuccess()
			}
			return
		}

		logger.Warn("Failed to join "+pool+" seeds, retrying",
			"error", err,
			"seeds", seeds,
			"attempt", attempt,
			"next_backoff", backoff,
		)

		select {
		case <-c.done:
			return
		case <-time.After(backoff):
		}

		backoff = min(backoff*2, 10*time.Second)
	}

	logger.Error("Exhausted retries joining "+pool+" seeds",
		"seeds", seeds,
		"attempts", maxJoinAttempts,
	)
}

// triggerEvalGateway sends a non-blocking signal to the gateway evaluator goroutine.
func (c *gossipCluster) triggerEvalGateway() {
	select {
	case c.evalCh <- struct{}{}:
	default:
		// Already pending evaluation
	}
}

// gatewayEvalLoop runs in a goroutine and processes gateway evaluation requests.
func (c *gossipCluster) gatewayEvalLoop() {
	for {
		select {
		case <-c.done:
			return
		case <-c.evalCh:
			c.evaluateGateway()
		}
	}
}

// Broadcast queues a message for delivery to all cluster members.
// The message is broadcast on the LAN pool. If this node is the gateway,
// it is also broadcast on the WAN pool.
func (c *gossipCluster) Broadcast(payload clusterv1.IsClusterMessage_Payload) error {
	msg := &clusterv1.ClusterMessage{
		Payload:      payload,
		SourceRegion: c.config.Region,
		SenderNode:   c.config.NodeID,
		SentAtMs:     time.Now().UnixMilli(),
	}

	c.mu.RLock()
	lanQ := c.lanQueue
	isGW := c.isGateway
	wanQ := c.wanQueue
	c.mu.RUnlock()

	if lanQ != nil {
		msg.Direction = clusterv1.Direction_DIRECTION_LAN
		lanBytes, err := proto.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal LAN message: %w", err)
		}
		lanQ.QueueBroadcast(newBroadcast(lanBytes))
	}

	if isGW && wanQ != nil {
		msg.Direction = clusterv1.Direction_DIRECTION_WAN
		wanBytes, err := proto.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal WAN message: %w", err)
		}
		wanQ.QueueBroadcast(newBroadcast(wanBytes))
	}

	return nil
}

// IsGateway returns whether this node is currently the WAN gateway.
func (c *gossipCluster) IsGateway() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isGateway
}

// WANAddr returns the WAN pool's advertise address (e.g. "127.0.0.1:54321")
// if this node is the gateway, or an empty string otherwise.
func (c *gossipCluster) WANAddr() string {
	c.mu.RLock()
	wan := c.wan
	c.mu.RUnlock()

	if wan == nil {
		return ""
	}

	return wan.LocalNode().FullAddress().Addr
}

// Members returns the current LAN memberlist nodes.
func (c *gossipCluster) Members() []*memberlist.Node {
	c.mu.RLock()
	lan := c.lan
	c.mu.RUnlock()

	if lan == nil {
		return nil
	}

	return lan.Members()
}

// Close gracefully leaves both LAN and WAN pools and shuts down.
// The closing flag prevents evaluateGateway from running during Leave.
func (c *gossipCluster) Close() error {
	c.closing.Store(true)
	close(c.done)

	// Demote from gateway first (leaves WAN).
	c.demoteFromGateway()

	// Grab the LAN memberlist reference then nil it under lock.
	c.mu.Lock()
	lan := c.lan
	c.lan = nil
	c.lanQueue = nil
	c.mu.Unlock()

	// Leave and shutdown without holding mu, since Leave triggers
	// NotifyLeave callbacks.
	if lan != nil {
		if err := lan.Leave(5 * time.Second); err != nil {
			logger.Warn("Error leaving LAN pool", "error", err)
		}

		if err := lan.Shutdown(); err != nil {
			return fmt.Errorf("failed to shutdown LAN memberlist: %w", err)
		}
	}

	return nil
}
