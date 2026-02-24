package cluster

import (
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/unkeyed/unkey/pkg/cluster/metrics"
	"github.com/unkeyed/unkey/pkg/logger"
)

// evaluateBridge checks whether this node should be the bridge.
// The node with the lexicographically smallest name wins.
func (c *gossipCluster) evaluateBridge() {
	// Don't evaluate during shutdown to avoid deadlocks
	if c.closing.Load() {
		return
	}

	c.mu.RLock()
	lan := c.lan
	c.mu.RUnlock()

	if lan == nil {
		return
	}

	members := lan.Members()
	if len(members) == 0 {
		return
	}

	// Find the node with the smallest name
	smallest := members[0]
	for _, m := range members[1:] {
		if m.Name < smallest.Name {
			smallest = m
		}
	}

	localName := lan.LocalNode().Name
	shouldBeBridge := smallest.Name == localName

	if shouldBeBridge && !c.IsBridge() {
		c.promoteToBridge()
	} else if !shouldBeBridge && c.IsBridge() {
		c.demoteFromBridge()
	}
}

// promoteToBridge creates a WAN memberlist and joins WAN seeds.
// memberlist.Create is performed outside the lock because it does network I/O
// and must not block Broadcast/Members/IsBridge. Retries with exponential
// backoff handle the case where a previous WAN memberlist's socket hasn't been
// fully released by the OS yet (e.g. after a rapid demote→promote cycle).
func (c *gossipCluster) promoteToBridge() {
	c.mu.Lock()
	if c.isBridge {
		c.mu.Unlock()
		return
	}

	logger.Info("Promoting to bridge", "node", c.config.NodeID, "region", c.config.Region)

	wanCfg := memberlist.DefaultWANConfig()
	wanCfg.Name = c.config.NodeID + "-wan"
	wanCfg.BindAddr = c.config.BindAddr
	wanCfg.BindPort = c.config.WANBindPort
	wanCfg.AdvertisePort = c.config.WANBindPort
	if c.config.WANAdvertiseAddr != "" {
		wanCfg.AdvertiseAddr = c.config.WANAdvertiseAddr
	}
	wanCfg.LogOutput = newLogWriter("wan")
	wanCfg.SecretKey = c.config.SecretKey
	wanCfg.Delegate = newWANDelegate(c)

	seeds := c.config.WANSeeds
	c.mu.Unlock()

	backoff := initialBackoff
	var wanList *memberlist.Memberlist

	for {
		if c.closing.Load() {
			return
		}

		var err error
		wanList, err = memberlist.Create(wanCfg)
		if err == nil {
			break
		}

		logger.Warn("Failed to create WAN memberlist, retrying",
			"error", err,
			"next_backoff", backoff,
		)

		select {
		case <-c.done:
			return
		case <-time.After(backoff):
		}

		backoff = min(backoff*2, reconnectInterval)
	}

	c.mu.Lock()
	if c.closing.Load() {
		c.mu.Unlock()
		wanList.Leave(5 * time.Second)  //nolint:errcheck
		wanList.Shutdown()              //nolint:errcheck
		return
	}

	c.wan = wanList
	c.wanQueue = &memberlist.TransmitLimitedQueue{
		NumNodes:       func() int { return wanList.NumMembers() },
		RetransmitMult: 4,
	}
	c.isBridge = true
	c.mu.Unlock()

	metrics.ClusterBridgeStatus.Set(1)
	metrics.ClusterBridgeTransitionsTotal.WithLabelValues("promoted").Inc()

	// Start background reconnection loop for WAN seeds.
	if len(seeds) > 0 {
		go c.maintainMembership("WAN", func() *memberlist.Memberlist {
			c.mu.RLock()
			defer c.mu.RUnlock()
			return c.wan
		}, seeds, nil)
	}
}

// demoteFromBridge shuts down the WAN memberlist.
func (c *gossipCluster) demoteFromBridge() {
	c.mu.Lock()
	if !c.isBridge {
		c.mu.Unlock()
		return
	}

	logger.Info("Demoting from bridge",
		"node", c.config.NodeID,
		"region", c.config.Region,
	)

	wan := c.wan
	c.wan = nil
	c.wanQueue = nil
	c.isBridge = false
	c.mu.Unlock()

	metrics.ClusterBridgeStatus.Set(0)
	metrics.ClusterBridgeTransitionsTotal.WithLabelValues("demoted").Inc()
	metrics.ClusterMembersCount.WithLabelValues("wan", c.config.Region).Set(0)

	// Leave and shutdown outside the lock since Leave can trigger callbacks
	if wan != nil {
		if err := wan.Leave(5 * time.Second); err != nil {
			logger.Warn("Error leaving WAN pool", "error", err)
		}

		if err := wan.Shutdown(); err != nil {
			logger.Warn("Error shutting down WAN memberlist", "error", err)
		}
	}
}
