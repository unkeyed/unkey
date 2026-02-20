package cluster

import (
	"github.com/hashicorp/memberlist"
	clusterv1 "github.com/unkeyed/unkey/gen/proto/cluster/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"google.golang.org/protobuf/proto"
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

	var msg clusterv1.ClusterMessage
	if err := proto.Unmarshal(data, &msg); err != nil {
		logger.Warn("Failed to unmarshal LAN cluster message", "error", err)
		return
	}

	// Deliver to the application callback
	if d.cluster.config.OnMessage != nil {
		d.cluster.config.OnMessage(&msg)
	}

	// If this node is the ambassador and the message originated locally (LAN direction),
	// relay it to the WAN pool for cross-region delivery.
	if d.cluster.IsAmbassador() && msg.Direction == clusterv1.Direction_DIRECTION_LAN {
		d.cluster.mu.RLock()
		wanQ := d.cluster.wanQueue
		d.cluster.mu.RUnlock()

		if wanQ != nil {
			relay := proto.Clone(&msg).(*clusterv1.ClusterMessage)
			relay.Direction = clusterv1.Direction_DIRECTION_WAN
			wanBytes, err := proto.Marshal(relay)
			if err != nil {
				logger.Warn("Failed to marshal WAN relay message", "error", err)
				return
			}
			wanQ.QueueBroadcast(newBroadcast(wanBytes))
		}
	}
}

// lanEventDelegate handles join/leave events for ambassador election.
type lanEventDelegate struct {
	cluster *gossipCluster
}

var _ memberlist.EventDelegate = (*lanEventDelegate)(nil)

func newLANEventDelegate(c *gossipCluster) *lanEventDelegate {
	return &lanEventDelegate{cluster: c}
}

func (d *lanEventDelegate) NotifyJoin(node *memberlist.Node) {
	d.cluster.triggerEvalAmbassador()
}

func (d *lanEventDelegate) NotifyLeave(node *memberlist.Node) {
	d.cluster.triggerEvalAmbassador()
}

func (d *lanEventDelegate) NotifyUpdate(node *memberlist.Node) {}
