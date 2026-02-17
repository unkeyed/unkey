package cluster

import (
	"github.com/hashicorp/memberlist"
	clusterv1 "github.com/unkeyed/unkey/gen/proto/cluster/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"google.golang.org/protobuf/proto"
)

// wanDelegate handles memberlist callbacks for the WAN pool.
type wanDelegate struct {
	cluster *gossipCluster
}

var _ memberlist.Delegate = (*wanDelegate)(nil)

func newWANDelegate(c *gossipCluster) *wanDelegate {
	return &wanDelegate{cluster: c}
}

func (d *wanDelegate) NodeMeta(limit int) []byte              { return nil }
func (d *wanDelegate) LocalState(join bool) []byte            { return nil }
func (d *wanDelegate) MergeRemoteState(buf []byte, join bool) {}
func (d *wanDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	d.cluster.mu.RLock()
	wanQ := d.cluster.wanQueue
	d.cluster.mu.RUnlock()

	if wanQ == nil {
		return nil
	}
	return wanQ.GetBroadcasts(overhead, limit)
}

// NotifyMsg is called when a message is received via the WAN pool.
func (d *wanDelegate) NotifyMsg(data []byte) {
	if len(data) == 0 {
		return
	}

	var msg clusterv1.ClusterMessage
	if err := proto.Unmarshal(data, &msg); err != nil {
		logger.Warn("Failed to unmarshal WAN cluster message", "error", err)
		return
	}

	// Skip messages that originated in our own region to avoid loops.
	if msg.SourceRegion == d.cluster.config.Region {
		return
	}

	// Deliver to the application callback on this bridge node
	if d.cluster.config.OnMessage != nil {
		d.cluster.config.OnMessage(&msg)
	}

	// Re-broadcast to the local LAN pool so all nodes in this region receive it.
	d.cluster.mu.RLock()
	lanQ := d.cluster.lanQueue
	d.cluster.mu.RUnlock()

	if lanQ == nil {
		return
	}

	msg.Direction = clusterv1.Direction_DIRECTION_WAN
	lanBytes, err := proto.Marshal(&msg)
	if err != nil {
		logger.Warn("Failed to marshal LAN relay message", "error", err)
		return
	}
	lanQ.QueueBroadcast(newBroadcast(lanBytes))
}
