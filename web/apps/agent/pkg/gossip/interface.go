package gossip

import (
	"context"
	"crypto/sha256"
)

type Member struct {
	NodeId  string
	RpcAddr string
}

// Hash returns a hash of the member to detect duplicates or changes.
func (m Member) Hash() []byte {
	h := sha256.New()
	h.Write([]byte(m.NodeId))
	h.Write([]byte(m.RpcAddr))
	return h.Sum(nil)
}

type Cluster interface {
	SubscribeJoinEvents(callerName string) <-chan Member
	SubscribeLeaveEvents(callerName string) <-chan Member
	Join(ctx context.Context, addrs ...string) error
	Shutdown(ctx context.Context) error
}
