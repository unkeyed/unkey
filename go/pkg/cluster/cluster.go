package cluster

import (
	"context"
	"fmt"

	"github.com/axiomhq/axiom-go/internal/config"
	"github.com/unkeyed/unkey/go/pkg/logging"
	"github.com/unkeyed/unkey/go/pkg/membership"
	"github.com/unkeyed/unkey/go/pkg/ring"
)

type Config struct {
	Self       Node
	Membership membership.Membership
	Logger     logging.Logger
}

func New(config config.Config) (*cluster, error) {

	r, err := ring.New[Node](ring.Config{
		TokensPerNode: 256,
		Logger:        config.Logger,
	})

	if err != nil {
		return nil, fmt.Errorf("unable to create hash ring: %w", err)
	}

	c := &cluster{
		self:       config.Self,
		membership: config.Membership,
		ring:       r,
		logger:     config.Logger,
	}

	return c, nil
}

type cluster struct {
	self       Node
	membership membership.Membership
	ring       *ring.Ring[Node]
	logger     logging.Logger
}

func (c *cluster) FindNode(ctx context.Context, key string) (Node, error) {
	node, err := c.ring.FindNode(key)
	if err != nil {
		return Node{}, err
	}
	return node.Tags, nil

}

func (c *cluster) Shutdown(ctx context.Context) error {
	return c.membership.Leave(ctx)

}
