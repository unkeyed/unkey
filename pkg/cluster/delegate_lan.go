package cluster

import (
	"github.com/hashicorp/memberlist"
)

// lanDelegate handles memberlist callbacks for the LAN pool.
type lanDelegate struct {
	cluster *gossipCluster
}

var _ memberlist.Delegate = (*lanDelegate)(nil)

func newLANDelegate(c *gossipCluster) *lanDelegate {
	return &lanDelegate{cluster: c}
}

func (d *lanDelegate) NodeMeta(limit int) []byte              { return nil }
func (d *lanDelegate) LocalState(join bool) []byte            { return nil }
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
	if d.cluster.IsGateway() && dir == dirLAN {
		d.cluster.mu.RLock()
		wanQ := d.cluster.wanQueue
		d.cluster.mu.RUnlock()

		if wanQ != nil {
			wanMsg := encodeMessage(dirWAN, sourceRegion, payload)
			wanQ.QueueBroadcast(newBroadcast(wanMsg))
		}
	}
}

// lanEventDelegate handles join/leave events for gateway election.
type lanEventDelegate struct {
	cluster *gossipCluster
}

var _ memberlist.EventDelegate = (*lanEventDelegate)(nil)

func newLANEventDelegate(c *gossipCluster) *lanEventDelegate {
	return &lanEventDelegate{cluster: c}
}

func (d *lanEventDelegate) NotifyJoin(node *memberlist.Node) {
	d.cluster.triggerEvalGateway()
}

func (d *lanEventDelegate) NotifyLeave(node *memberlist.Node) {
	d.cluster.triggerEvalGateway()
}

func (d *lanEventDelegate) NotifyUpdate(node *memberlist.Node) {}
