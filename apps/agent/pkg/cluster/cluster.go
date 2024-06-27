package cluster

import (
	"fmt"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/membership"
	"github.com/unkeyed/unkey/apps/agent/pkg/ring"
)

type cluster struct {
	id         string
	membership membership.Membership
	logger     logging.Logger

	// The hash ring is used to determine which node is responsible for a given key.
	ring *ring.Ring[Node]

	shutdownCh chan struct{}
	// bearer token used to authenticate with other nodes
	authToken string
}

type Config struct {
	NodeId     string
	Membership membership.Membership
	Logger     logging.Logger
	Debug      bool
	RpcAddr    string
	AuthToken  string
}

func New(config Config) (Cluster, error) {

	r, err := ring.New[Node](ring.Config{
		TokensPerNode: 256,
		Logger:        config.Logger,
	})
	if err != nil {
		return nil, err
	}

	c := &cluster{
		id:         config.NodeId,
		membership: config.Membership,
		logger:     config.Logger,
		ring:       r,
		shutdownCh: make(chan struct{}),
		authToken:  config.AuthToken,
	}

	go func() {
		joins := c.membership.SubscribeJoinEvents()
		leaves := c.membership.SubscribeLeaveEvents()
		for {
			select {
			case join := <-joins:
				c.logger.Info().Str("node", join.Id).Msg("adding node to ring")
				err = r.AddNode(ring.Node[Node]{
					Id:   join.Id,
					Tags: Node{Id: join.Id, RpcAddr: join.RpcAddr},
				})
				if err != nil {
					c.logger.Error().Err(err).Str("nodeId", join.Id).Msg("unable to add node to ring")
				}
			case leave := <-leaves:
				c.logger.Info().Str("node", leave.Id).Msg("removing node from ring")
				err := r.RemoveNode(leave.Id)
				if err != nil {
					c.logger.Error().Err(err).Str("nodeId", leave.Id).Msg("unable to remove node from ring")
				}
			}
		}
	}()

	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-c.shutdownCh:
				return
			case <-t.C:
				members, err := c.membership.Members()
				if err != nil {
					c.logger.Error().Err(err).Msg("failed to get members")
					continue
				}
				memberAddrs := make([]string, len(members))
				for i, member := range members {
					memberAddrs[i] = member.Id
				}
				c.logger.Info().Int("clusterSize", len(members)).Str("nodeId", c.id).Send()
			}
		}
	}()

	return c, nil

}
func (c *cluster) NodeId() string {
	return c.id
}

func (c *cluster) AuthToken() string {
	return c.authToken
}

func (c *cluster) FindNodes(key string, n int) ([]Node, error) {
	found, err := c.ring.FindNodes(key, n)
	if err != nil {
		return nil, fmt.Errorf("failed to find nodes: %w", err)
	}

	nodes := make([]Node, len(found))
	for i, r := range found {
		nodes[i] = r.Tags
	}
	return nodes, nil

}
func (c *cluster) Join(addrs []string) (clusterSize int, err error) {
	addrsWithoutSelf := []string{}
	for _, addr := range addrs {
		if addr != c.membership.Addr() {
			addrsWithoutSelf = append(addrsWithoutSelf, addr)
		}
	}
	members, err := c.membership.Join(addrsWithoutSelf...)
	if err != nil {
		return 0, fmt.Errorf("failed to join serf cluster: %w", err)
	}
	return members, nil
}

func (c *cluster) FindNode(key string) (Node, error) {
	found, err := c.ring.FindNodes(key, 1)
	if err != nil {
		return Node{}, fmt.Errorf("failed to find node: %w", err)
	}
	if len(found) == 0 {
		return Node{}, fmt.Errorf("no nodes found")
	}
	return found[0].Tags, nil

}

func (c *cluster) Shutdown() error {
	close(c.shutdownCh)
	return c.membership.Leave()
}
