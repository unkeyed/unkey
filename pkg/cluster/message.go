package cluster

import (
	"github.com/hashicorp/memberlist"
)

// clusterBroadcast implements memberlist.Broadcast for the TransmitLimitedQueue.
type clusterBroadcast struct {
	msg []byte
}

var _ memberlist.Broadcast = (*clusterBroadcast)(nil)

func (b *clusterBroadcast) Invalidates(other memberlist.Broadcast) bool { return false }
func (b *clusterBroadcast) Message() []byte                             { return b.msg }
func (b *clusterBroadcast) Finished()                                   {}

// newBroadcast wraps raw bytes in a memberlist.Broadcast for queue submission.
func newBroadcast(msg []byte) *clusterBroadcast {
	return &clusterBroadcast{msg: msg}
}
