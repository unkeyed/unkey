package cluster

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/memberlist"
	clusterv1 "github.com/unkeyed/unkey/gen/proto/cluster/v1"
	"github.com/unkeyed/unkey/pkg/cluster/metrics"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/repeat"
	"google.golang.org/protobuf/proto"
)

const (
	// initialBackoff is the starting delay for exponential backoff retries.
	initialBackoff = 500 * time.Millisecond

	// reconnectInterval is how often the background loop checks whether
	// the node is isolated and needs to re-join seeds.
	reconnectInterval = 30 * time.Second
)

// Cluster is the public interface for gossip-based cluster membership.
type Cluster interface {
	Broadcast(msg clusterv1.IsClusterMessage_Payload) error
	Members() []*memberlist.Node
	IsBridge() bool
	WANAddr() string
	Close() error
}

// gossipCluster manages a two-tier gossip membership: a LAN pool for intra-region
// communication and, on the elected bridge node, a WAN pool for cross-region
// communication.
type gossipCluster struct {
	config Config

	mu           sync.RWMutex
	lan          *memberlist.Memberlist
	lanQueue     *memberlist.TransmitLimitedQueue
	wan          *memberlist.Memberlist
	wanQueue     *memberlist.TransmitLimitedQueue
	isBridge bool
	closing      atomic.Bool

	// evalCh is used to trigger async bridge evaluation from memberlist
	// callbacks. This avoids calling Members() inside NotifyJoin/NotifyLeave
	// where memberlist holds its internal state lock.
	evalCh chan struct{}
	done   chan struct{}

	// stopMetrics stops the periodic member count gauge updater.
	stopMetrics func()
}

// New creates a new cluster node, starts the LAN memberlist, joins LAN seeds,
// and begins bridge evaluation.
func New(cfg Config) (Cluster, error) {
	cfg.setDefaults()

	c := &gossipCluster{
		config:       cfg,
		mu:           sync.RWMutex{},
		lan:          nil,
		lanQueue:     nil,
		wan:          nil,
		wanQueue:     nil,
		isBridge: false,
		closing:      atomic.Bool{},
		evalCh:       make(chan struct{}, 1),
		done:         make(chan struct{}),
		stopMetrics:  nil, // set below
	}

	// Start the async bridge evaluator
	go c.bridgeEvalLoop()

	// Configure LAN memberlist
	lanCfg := memberlist.DefaultLANConfig()
	lanCfg.Name = cfg.NodeID
	lanCfg.BindAddr = cfg.BindAddr
	lanCfg.BindPort = cfg.BindPort
	lanCfg.AdvertisePort = cfg.BindPort
	lanCfg.LogOutput = newLogWriter("lan")
	lanCfg.SecretKey = cfg.SecretKey
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

	// Start background reconnection loop for LAN seeds. This handles both
	// initial join (DNS may not be ready at startup) and reconnection after
	// network partitions or rolling restarts of other nodes.
	if len(cfg.LANSeeds) > 0 {
		go c.maintainMembership("LAN", func() *memberlist.Memberlist {
			c.mu.RLock()
			defer c.mu.RUnlock()
			return c.lan
		}, cfg.LANSeeds, c.triggerEvalBridge)
	}

	// Periodically update pool member count gauges. This avoids tracking
	// counts inside memberlist callbacks where internal locks are held.
	c.stopMetrics = repeat.Every(1*time.Minute, func() {
		c.mu.RLock()
		lan := c.lan
		wan := c.wan
		c.mu.RUnlock()

		if lan != nil {
			metrics.ClusterMembersCount.WithLabelValues("lan", c.config.Region).Set(float64(lan.NumMembers()))
		}
		if wan != nil {
			metrics.ClusterMembersCount.WithLabelValues("wan", c.config.Region).Set(float64(wan.NumMembers()))
		}
	})

	// Trigger initial bridge evaluation
	c.triggerEvalBridge()

	return c, nil
}

// maintainMembership runs for the lifetime of the cluster and ensures the node
// stays connected to seeds. It handles initial join (with backoff for DNS
// readiness) and periodic reconnection if the node becomes isolated.
func (c *gossipCluster) maintainMembership(pool string, list func() *memberlist.Memberlist, seeds []string, onJoin func()) {
	backoff := initialBackoff

	for {
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
			metrics.ClusterSeedJoinAttemptsTotal.WithLabelValues(pool, "success").Inc()
			logger.Info("Joined "+pool+" seeds", "seeds", seeds)
			if onJoin != nil {
				onJoin()
			}
			break
		}

		metrics.ClusterSeedJoinAttemptsTotal.WithLabelValues(pool, "failure").Inc()
		logger.Warn("Failed to join "+pool+" seeds, retrying",
			"error", err,
			"seeds", seeds,
			"next_backoff", backoff,
		)

		select {
		case <-c.done:
			return
		case <-time.After(backoff):
		}

		backoff = min(backoff*2, reconnectInterval)
	}

	// Background loop — periodically check if we're isolated and re-join.
	ticker := time.NewTicker(reconnectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
		}

		ml := list()
		if ml == nil {
			return
		}

		// If we have more than just ourselves, we're connected.
		if ml.NumMembers() > 1 {
			continue
		}

		// We're alone — try to rejoin seeds.
		logger.Warn("Node is isolated, attempting to rejoin "+pool+" seeds",
			"seeds", seeds,
		)

		_, err := ml.Join(seeds)
		if err != nil {
			metrics.ClusterSeedJoinAttemptsTotal.WithLabelValues(pool, "failure").Inc()
			logger.Warn("Failed to rejoin "+pool+" seeds",
				"error", err,
				"seeds", seeds,
			)
			continue
		}

		metrics.ClusterSeedJoinAttemptsTotal.WithLabelValues(pool, "success").Inc()
		logger.Info("Rejoined "+pool+" seeds", "seeds", seeds)
		if onJoin != nil {
			onJoin()
		}
	}
}

// triggerEvalBridge sends a non-blocking signal to the bridge evaluator goroutine.
func (c *gossipCluster) triggerEvalBridge() {
	select {
	case c.evalCh <- struct{}{}:
	default:
		// Already pending evaluation
	}
}

// bridgeEvalLoop runs in a goroutine and processes bridge evaluation requests.
func (c *gossipCluster) bridgeEvalLoop() {
	for {
		select {
		case <-c.done:
			return
		case <-c.evalCh:
			c.evaluateBridge()
		}
	}
}

// Broadcast queues a message for delivery to all cluster members.
// The message is broadcast on the LAN pool. If this node is the bridge,
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
	isBr := c.isBridge
	wanQ := c.wanQueue
	c.mu.RUnlock()

	if lanQ != nil {
		msg.Direction = clusterv1.Direction_DIRECTION_LAN
		lanBytes, err := proto.Marshal(msg)
		if err != nil {
			metrics.ClusterBroadcastErrorsTotal.WithLabelValues("lan").Inc()
			return fmt.Errorf("failed to marshal LAN message: %w", err)
		}
		lanQ.QueueBroadcast(newBroadcast(lanBytes))
		metrics.ClusterBroadcastsTotal.WithLabelValues("lan").Inc()
	}

	if isBr && wanQ != nil {
		msg.Direction = clusterv1.Direction_DIRECTION_WAN
		wanBytes, err := proto.Marshal(msg)
		if err != nil {
			metrics.ClusterBroadcastErrorsTotal.WithLabelValues("wan").Inc()
			return fmt.Errorf("failed to marshal WAN message: %w", err)
		}
		wanQ.QueueBroadcast(newBroadcast(wanBytes))
		metrics.ClusterBroadcastsTotal.WithLabelValues("wan").Inc()
	}

	return nil
}

// IsBridge returns whether this node is currently the WAN bridge.
func (c *gossipCluster) IsBridge() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isBridge
}

// WANAddr returns the WAN pool's advertise address (e.g. "127.0.0.1:54321")
// if this node is the bridge, or an empty string otherwise.
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
// The closing flag prevents evaluateBridge from running during Leave.
// Safe to call multiple times; only the first call performs the shutdown.
func (c *gossipCluster) Close() error {
	if alreadyClosing := c.closing.Swap(true); alreadyClosing {
		return nil
	}
	close(c.done)

	if c.stopMetrics != nil {
		c.stopMetrics()
	}

	// Demote from bridge first (leaves WAN).
	c.demoteFromBridge()

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
