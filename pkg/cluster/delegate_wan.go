package cluster

import (
	"github.com/hashicorp/memberlist"
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

	_, sourceRegion, payload, err := decodeMessage(data)
	if err != nil {
		return
	}

	// Skip messages that originated in our own region to avoid loops.
	if sourceRegion == d.cluster.config.Region {
		return
	}

	// Deliver to the application callback on this gateway node
	if d.cluster.config.OnMessage != nil {
		d.cluster.config.OnMessage(payload)
	}

	// Re-broadcast to the local LAN pool so all nodes in this region receive it.
	lanMsg := encodeMessage(dirWAN, sourceRegion, payload)
	d.cluster.lanQueue.QueueBroadcast(newBroadcast(lanMsg))
}
