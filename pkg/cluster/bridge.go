package cluster

import (
	"io"
	"time"

	"github.com/hashicorp/memberlist"
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
	wanCfg.LogOutput = io.Discard
	wanCfg.SecretKey = c.config.SecretKey

	wanCfg.Delegate = newWANDelegate(c)

	wanList, err := memberlist.Create(wanCfg)
	if err != nil {
		c.mu.Unlock()
		logger.Error("Failed to create WAN memberlist", "error", err)
		return
	}

	c.wan = wanList
	c.wanQueue = &memberlist.TransmitLimitedQueue{
		NumNodes:       func() int { return wanList.NumMembers() },
		RetransmitMult: 4,
	}

	c.isBridge = true
	seeds := c.config.WANSeeds
	c.mu.Unlock()

	// Join WAN seeds outside the lock with retries
	if len(seeds) > 0 {
		go c.joinSeeds("WAN", func() *memberlist.Memberlist {
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
