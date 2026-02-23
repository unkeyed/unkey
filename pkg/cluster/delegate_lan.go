package cluster

import (
	"time"

	"github.com/hashicorp/memberlist"
	clusterv1 "github.com/unkeyed/unkey/gen/proto/cluster/v1"
	"github.com/unkeyed/unkey/pkg/cluster/metrics"
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
		metrics.ClusterMessageUnmarshalErrorsTotal.WithLabelValues("lan").Inc()
		logger.Warn("Failed to unmarshal LAN cluster message", "error", err)
		return
	}

	direction := "lan"
	if msg.Direction == clusterv1.Direction_DIRECTION_WAN {
		direction = "wan"
	}
	payloadType := metrics.PayloadTypeName(msg.GetPayload())
	metrics.ClusterMessagesReceivedTotal.WithLabelValues("lan", direction, payloadType).Inc()

	if msg.SentAtMs > 0 {
		latency := time.Since(time.UnixMilli(msg.SentAtMs)).Seconds()
		if latency >= 0 {
			metrics.ClusterMessageLatencySeconds.WithLabelValues(direction, msg.SourceRegion).Observe(latency)
		}
	}

	// Deliver to the application callback
	if d.cluster.config.OnMessage != nil {
		d.cluster.config.OnMessage(&msg)
	}

	// If this node is the bridge and the message originated locally (LAN direction),
	// relay it to the WAN pool for cross-region delivery.
	if d.cluster.IsBridge() && msg.Direction == clusterv1.Direction_DIRECTION_LAN {
		d.cluster.mu.RLock()
		wanQ := d.cluster.wanQueue
		d.cluster.mu.RUnlock()

		if wanQ != nil {
			relay := proto.Clone(&msg).(*clusterv1.ClusterMessage)
			relay.Direction = clusterv1.Direction_DIRECTION_WAN
			wanBytes, err := proto.Marshal(relay)
			if err != nil {
				metrics.ClusterRelayErrorsTotal.WithLabelValues("lan_to_wan").Inc()
				logger.Warn("Failed to marshal WAN relay message", "error", err)
				return
			}
			wanQ.QueueBroadcast(newBroadcast(wanBytes))
			metrics.ClusterRelaysTotal.WithLabelValues("lan_to_wan").Inc()
		}
	}
}

// lanEventDelegate handles join/leave events for bridge election.
type lanEventDelegate struct {
	cluster *gossipCluster
}

var _ memberlist.EventDelegate = (*lanEventDelegate)(nil)

func newLANEventDelegate(c *gossipCluster) *lanEventDelegate {
	return &lanEventDelegate{cluster: c}
}

func (d *lanEventDelegate) NotifyJoin(node *memberlist.Node) {
	metrics.ClusterMembershipEventsTotal.WithLabelValues("join").Inc()
	d.cluster.triggerEvalBridge()
}

func (d *lanEventDelegate) NotifyLeave(node *memberlist.Node) {
	metrics.ClusterMembershipEventsTotal.WithLabelValues("leave").Inc()
	d.cluster.triggerEvalBridge()
}

func (d *lanEventDelegate) NotifyUpdate(node *memberlist.Node) {}
