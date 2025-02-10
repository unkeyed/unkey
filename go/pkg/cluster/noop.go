package cluster

import (
	"context"
	"net"
)

type noop struct {
	self Node
}

var _ Cluster = (*noop)(nil)

func NewNoop(id string, addr net.IP) *noop {
	return &noop{self: Node{
		ID:      id,
		Addr:    addr,
		RpcAddr: "",
	}}
}

func (n *noop) Self() Node {
	return n.self
}

func (n *noop) FindNode(ctx context.Context, key string) (Node, error) {
	return n.self, nil

}
func (n *noop) Shutdown(ctx context.Context) error {
	return nil
}

func (n *noop) SubscribeJoin() <-chan Node {
	return make(chan Node)
}

func (n *noop) SubscribeLeave() <-chan Node {
	return make(chan Node)
}
