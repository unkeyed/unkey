package cluster

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/bufbuild/connect-go"
	clusterv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/cluster/v1"
	"github.com/unkeyed/unkey/apps/agent/gen/proto/cluster/v1/clusterv1connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/membership"
	"github.com/unkeyed/unkey/apps/agent/pkg/repeat"
	"github.com/unkeyed/unkey/apps/agent/pkg/ring"
)

type cluster struct {
	id         string
	membership membership.Membership
	logger     logging.Logger

	// The hash ring is used to determine which node is responsible for a given key.
	ring *ring.Ring[Node]

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

	repeat.Every(10*time.Second, func() {
		members, err := c.membership.Members()
		if err != nil {
			c.logger.Error().Err(err).Msg("failed to get members")
			return
		}
		memberAddrs := make([]string, len(members))
		for i, member := range members {
			memberAddrs[i] = member.Id
		}
		c.logger.Info().Int("clusterSize", len(members)).Str("nodeId", c.id).Send()
	})

	return c, nil

}

func (c *cluster) SyncMembership() error {
	return c.membership.Sync()
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
	c.logger.Info().Msg("shutting down cluster")

	members, err := c.membership.Members()
	if err != nil {
		return fmt.Errorf("failed to get members: %w", err)

	}

	err = c.membership.Leave()
	if err != nil {
		return fmt.Errorf("failed to leave membership: %w", err)
	}

	ctx := context.Background()
	wg := sync.WaitGroup{}
	for _, m := range members {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req := connect.NewRequest(&clusterv1.AnnounceStateChangeRequest{
				NodeId: c.id,
				State:  clusterv1.NodeState_NODE_STATE_LEAVING,
			})
			req.Header().Set("Authorization", c.authToken)

			_, err := clusterv1connect.NewClusterServiceClient(http.DefaultClient, m.RpcAddr).AnnounceStateChange(ctx, req)
			if err != nil {
				c.logger.Error().Err(err).Str("peerId", m.Id).Msg("failed to announce state change")
			}
		}()
	}
	wg.Wait()
	return nil
}
