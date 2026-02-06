package cluster

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/unkeyed/unkey/pkg/logger"
)

// nodeNameWithTimestamp encodes the node ID and join time into a memberlist node name.
// Format: "{nodeID}:{unixNano}"
func nodeNameWithTimestamp(nodeID string, joinTime time.Time) string {
	return fmt.Sprintf("%s:%d", nodeID, joinTime.UnixNano())
}

// parseJoinTime extracts the join time from a node name.
// Returns zero time if parsing fails.
func parseJoinTime(nodeName string) time.Time {
	idx := strings.LastIndex(nodeName, ":")
	if idx < 0 || idx == len(nodeName)-1 {
		return time.Time{}
	}

	nanos, err := strconv.ParseInt(nodeName[idx+1:], 10, 64)
	if err != nil {
		return time.Time{}
	}

	return time.Unix(0, nanos)
}

// evaluateGateway checks whether this node should be the gateway.
// The oldest node in the LAN pool (by join time encoded in node name) wins.
func (c *Cluster) evaluateGateway() {
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

	// Find the oldest node
	var oldest *memberlist.Node
	var oldestTime time.Time

	for _, m := range members {
		jt := parseJoinTime(m.Name)
		if jt.IsZero() {
			continue
		}
		if oldest == nil || jt.Before(oldestTime) {
			oldest = m
			oldestTime = jt
		}
	}

	if oldest == nil {
		return
	}

	localName := lan.LocalNode().Name
	shouldBeGateway := oldest.Name == localName

	if shouldBeGateway && !c.IsGateway() {
		c.promoteToGateway()
	} else if !shouldBeGateway && c.IsGateway() {
		c.demoteFromGateway()
	}
}

// promoteToGateway creates a WAN memberlist and joins WAN seeds.
func (c *Cluster) promoteToGateway() {
	c.mu.Lock()
	if c.isGateway {
		c.mu.Unlock()
		return
	}

	logger.Info("Promoting to gateway",
		"node", c.config.NodeID,
		"region", c.config.Region,
	)

	wanCfg := memberlist.DefaultWANConfig()
	wanCfg.Name = nodeNameWithTimestamp(c.config.NodeID+"-wan", c.joinTime)
	wanCfg.BindAddr = c.config.BindAddr
	wanCfg.BindPort = c.config.WANBindPort
	wanCfg.AdvertisePort = c.config.WANBindPort
	wanCfg.LogOutput = logger.NewMemberlistWriter()

	wanDel := &wanDelegate{cluster: c}
	wanCfg.Delegate = wanDel

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

	// Join WAN seeds outside the lock to avoid blocking
	if len(seeds) > 0 {
		go func() {
			_, joinErr := wanList.Join(seeds)
			if joinErr != nil {
				logger.Warn("Failed to join WAN seeds", "error", joinErr, "seeds", seeds)
			}
		}()
	}
}

// demoteFromGateway shuts down the WAN memberlist.
func (c *Cluster) demoteFromGateway() {
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
