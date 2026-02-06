package cluster

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/unkeyed/unkey/pkg/logger"
)

// Cluster manages a two-tier gossip membership: a LAN pool for intra-region
// communication and, on the elected gateway node, a WAN pool for cross-region
// communication.
type Cluster struct {
	config Config

	mu        sync.RWMutex
	lan       *memberlist.Memberlist
	lanQueue  *memberlist.TransmitLimitedQueue
	wan       *memberlist.Memberlist
	wanQueue  *memberlist.TransmitLimitedQueue
	isGateway bool
	joinTime  time.Time
	noop      bool
	closing   atomic.Bool

	// evalCh is used to trigger async gateway evaluation from memberlist
	// callbacks. This avoids calling Members() inside NotifyJoin/NotifyLeave
	// where memberlist holds its internal state lock.
	evalCh chan struct{}
	done   chan struct{}
}

// New creates a new cluster node, starts the LAN memberlist, joins LAN seeds,
// and begins gateway evaluation.
func New(cfg Config) (*Cluster, error) {
	cfg.setDefaults()

	now := time.Now()

	c := &Cluster{
		config:    cfg,
		mu:        sync.RWMutex{},
		lan:       nil,
		lanQueue:  nil,
		wan:       nil,
		wanQueue:  nil,
		isGateway: false,
		joinTime:  now,
		noop:      false,
		closing:   atomic.Bool{},
		evalCh:    make(chan struct{}, 1),
		done:      make(chan struct{}),
	}

	// Start the async gateway evaluator
	go c.gatewayEvalLoop()

	// Configure LAN memberlist
	lanCfg := memberlist.DefaultLANConfig()
	lanCfg.Name = nodeNameWithTimestamp(cfg.NodeID, now)
	lanCfg.BindAddr = cfg.BindAddr
	lanCfg.BindPort = cfg.BindPort
	lanCfg.AdvertisePort = cfg.BindPort
	lanCfg.LogOutput = logger.NewMemberlistWriter()

	lanDel := &lanDelegate{cluster: c}
	lanCfg.Delegate = lanDel
	lanCfg.Events = &lanEventDelegate{cluster: c}

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

	// Join LAN seeds
	if len(cfg.LANSeeds) > 0 {
		_, err = lan.Join(cfg.LANSeeds)
		if err != nil {
			logger.Warn("Failed to join LAN seeds",
				"error", err,
				"seeds", cfg.LANSeeds,
			)
		}
	}

	// Trigger initial gateway evaluation
	c.triggerEvalGateway()

	return c, nil
}

// triggerEvalGateway sends a non-blocking signal to the gateway evaluator goroutine.
func (c *Cluster) triggerEvalGateway() {
	select {
	case c.evalCh <- struct{}{}:
	default:
		// Already pending evaluation
	}
}

// gatewayEvalLoop runs in a goroutine and processes gateway evaluation requests.
func (c *Cluster) gatewayEvalLoop() {
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
func (c *Cluster) Broadcast(msg []byte) error {
	if c.noop {
		return nil
	}

	c.mu.RLock()
	lanQ := c.lanQueue
	isGW := c.isGateway
	wanQ := c.wanQueue
	c.mu.RUnlock()

	if lanQ != nil {
		lanMsg := encodeMessage(dirLAN, c.config.Region, msg)
		lanQ.QueueBroadcast(&clusterBroadcast{msg: lanMsg})
	}

	if isGW && wanQ != nil {
		wanMsg := encodeMessage(dirWAN, c.config.Region, msg)
		wanQ.QueueBroadcast(&clusterBroadcast{msg: wanMsg})
	}

	return nil
}

// IsGateway returns whether this node is currently the WAN gateway.
func (c *Cluster) IsGateway() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isGateway
}

// WANAddr returns the WAN pool's advertise address (e.g. "127.0.0.1:54321")
// if this node is the gateway, or an empty string otherwise.
func (c *Cluster) WANAddr() string {
	c.mu.RLock()
	wan := c.wan
	c.mu.RUnlock()

	if wan == nil {
		return ""
	}
	return wan.LocalNode().FullAddress().Addr
}

// Members returns the current LAN memberlist nodes.
func (c *Cluster) Members() []*memberlist.Node {
	if c.noop {
		return nil
	}
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
func (c *Cluster) Close() error {
	if c.noop {
		return nil
	}

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
