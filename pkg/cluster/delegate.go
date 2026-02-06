package cluster

import (
	"github.com/hashicorp/memberlist"
)

// lanDelegate handles memberlist callbacks for the LAN pool.
type lanDelegate struct {
	cluster *Cluster
}

var _ memberlist.Delegate = (*lanDelegate)(nil)

func (d *lanDelegate) NodeMeta(limit int) []byte { return nil }
func (d *lanDelegate) LocalState(join bool) []byte { return nil }
func (d *lanDelegate) MergeRemoteState(buf []byte, join bool) {}
func (d *lanDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	d.cluster.mu.RLock()
	q := d.cluster.lanQueue
	d.cluster.mu.RUnlock()

	if q == nil {
		return nil
	}
	return q.GetBroadcasts(overhead, limit)
}

// NotifyMsg is called when a message is received via the LAN pool.
func (d *lanDelegate) NotifyMsg(data []byte) {
	if len(data) == 0 {
		return
	}

	dir, sourceRegion, payload, err := decodeMessage(data)
	if err != nil {
		return
	}

	// Deliver to the application callback
	if d.cluster.config.OnMessage != nil {
		d.cluster.config.OnMessage(payload)
	}

	// If this node is the gateway and the message originated locally (LAN direction),
	// relay it to the WAN pool for cross-region delivery.
	if dir == dirLAN && d.cluster.IsGateway() {
		d.cluster.mu.RLock()
		wanQ := d.cluster.wanQueue
		d.cluster.mu.RUnlock()

		if wanQ != nil {
			wanMsg := encodeMessage(dirWAN, sourceRegion, payload)
			wanQ.QueueBroadcast(&clusterBroadcast{msg: wanMsg})
		}
	}
}

// wanDelegate handles memberlist callbacks for the WAN pool.
type wanDelegate struct {
	cluster *Cluster
}

var _ memberlist.Delegate = (*wanDelegate)(nil)

func (d *wanDelegate) NodeMeta(limit int) []byte { return nil }
func (d *wanDelegate) LocalState(join bool) []byte { return nil }
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
	d.cluster.lanQueue.QueueBroadcast(&clusterBroadcast{msg: lanMsg})
}

// lanEventDelegate handles join/leave events for gateway election.
type lanEventDelegate struct {
	cluster *Cluster
}

var _ memberlist.EventDelegate = (*lanEventDelegate)(nil)

func (d *lanEventDelegate) NotifyJoin(node *memberlist.Node) {
	d.cluster.triggerEvalGateway()
}

func (d *lanEventDelegate) NotifyLeave(node *memberlist.Node) {
	d.cluster.triggerEvalGateway()
}

func (d *lanEventDelegate) NotifyUpdate(node *memberlist.Node) {}
