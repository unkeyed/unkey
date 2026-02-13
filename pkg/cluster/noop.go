package cluster

import (
	"github.com/hashicorp/memberlist"
	clusterv1 "github.com/unkeyed/unkey/gen/proto/cluster/v1"
)

// noopCluster is a no-op implementation of Cluster that does not participate in gossip.
// All operations are safe to call but do nothing.
type noopCluster struct{}

var _ Cluster = noopCluster{}

func (noopCluster) Broadcast(clusterv1.IsClusterMessage_Payload) error { return nil }
func (noopCluster) Members() []*memberlist.Node               { return nil }
func (noopCluster) IsGateway() bool                           { return false }
func (noopCluster) WANAddr() string                           { return "" }
func (noopCluster) Close() error                              { return nil }

// NewNoop returns a no-op cluster that does not participate in gossip.
func NewNoop() Cluster {
	return noopCluster{}
}
