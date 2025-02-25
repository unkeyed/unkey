package cluster

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/unkeyed/unkey/go/pkg/events"
	"github.com/unkeyed/unkey/go/pkg/logging"
	"github.com/unkeyed/unkey/go/pkg/membership"
	"github.com/unkeyed/unkey/go/pkg/ring"
)

type Config struct {
	Self       Node
	Membership membership.Membership
	Logger     logging.Logger

	RpcPort int
}

func New(config Config) (*cluster, error) {

	r, err := ring.New[Node](ring.Config{
		TokensPerNode: 256,
		Logger:        config.Logger,
	})

	if err != nil {
		return nil, fmt.Errorf("unable to create hash ring: %w", err)
	}

	c := &cluster{
		self:        config.Self,
		membership:  config.Membership,
		ring:        r,
		logger:      config.Logger,
		joinEvents:  events.NewTopic[Node](),
		leaveEvents: events.NewTopic[Node](),

		rpcPort: config.RpcPort,
	}

	go c.keepInSync()

	return c, nil
}

type cluster struct {
	self       Node
	membership membership.Membership
	ring       *ring.Ring[Node]
	logger     logging.Logger

	joinEvents  events.Topic[Node]
	leaveEvents events.Topic[Node]

	rpcPort int
}

var _ Cluster = (*cluster)(nil)

func (c *cluster) Self() Node {
	return c.self
}

func (c *cluster) SubscribeJoin() <-chan Node {
	return c.joinEvents.Subscribe("cluster.joinEvents")
}

func (c *cluster) SubscribeLeave() <-chan Node {
	return c.leaveEvents.Subscribe("cluster.leaveEvents")
}

// listens to membership changes and updates the hash ring
func (c *cluster) keepInSync() {
	joins := c.membership.SubscribeJoinEvents()
	leaves := c.membership.SubscribeLeaveEvents()

	for {
		select {
		case node := <-joins:
			{
				ctx := context.Background()
				c.logger.Info(ctx, "node joined", slog.String("nodeID", node.NodeID))

				err := c.ring.AddNode(ctx, ring.Node[Node]{
					ID: node.NodeID,
					Tags: Node{
						RpcAddr: fmt.Sprintf("%s:%d", node.Addr, c.rpcPort),
						ID:      node.NodeID,
						Addr:    node.Addr,
					},
				})
				if err != nil {
					c.logger.Error(ctx, "failed to add node to ring", slog.String("error", err.Error()))
				}

			}
		case node := <-leaves:
			{
				ctx := context.Background()
				c.logger.Info(ctx, "node left", slog.String("nodeID", node.NodeID))
				err := c.ring.RemoveNode(ctx, node.NodeID)
				if err != nil {
					c.logger.Error(ctx, "failed to remove node from ring", slog.String("error", err.Error()))
				}
			}
		}

	}

}

func (c *cluster) FindNode(ctx context.Context, key string) (Node, error) {
	node, err := c.ring.FindNode(key)
	if err != nil {
		return Node{}, err
	}
	return node.Tags, nil

}

func (c *cluster) Shutdown(ctx context.Context) error {
	return c.membership.Leave()
}
