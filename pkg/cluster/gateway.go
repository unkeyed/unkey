package cluster

import (
	"io"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/unkeyed/unkey/pkg/logger"
)

// evaluateGateway checks whether this node should be the gateway.
// The node with the lexicographically smallest name wins.
func (c *gossipCluster) evaluateGateway() {
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
	shouldBeGateway := smallest.Name == localName

	if shouldBeGateway && !c.IsGateway() {
		c.promoteToGateway()
	} else if !shouldBeGateway && c.IsGateway() {
		c.demoteFromGateway()
	}
}

// promoteToGateway creates a WAN memberlist and joins WAN seeds.
func (c *gossipCluster) promoteToGateway() {
	c.mu.Lock()
	if c.isGateway {
		c.mu.Unlock()
		return
	}

	logger.Info("Promoting to gateway", "node", c.config.NodeID, "region", c.config.Region)

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

	c.isGateway = true
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

// demoteFromGateway shuts down the WAN memberlist.
func (c *gossipCluster) demoteFromGateway() {
	c.mu.Lock()
	if !c.isGateway {
		c.mu.Unlock()
		return
	}

	logger.Info("Demoting from gateway",
		"node", c.config.NodeID,
		"region", c.config.Region,
	)

	wan := c.wan
	c.wan = nil
	c.wanQueue = nil
	c.isGateway = false
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
