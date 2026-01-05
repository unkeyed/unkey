package cluster

import (
	"fmt"
	"time"

	"github.com/unkeyed/unkey/svc/agent/pkg/logging"
	"github.com/unkeyed/unkey/svc/agent/pkg/membership"
	"github.com/unkeyed/unkey/svc/agent/pkg/metrics"
	"github.com/unkeyed/unkey/svc/agent/pkg/prometheus"
	"github.com/unkeyed/unkey/svc/agent/pkg/repeat"
	"github.com/unkeyed/unkey/svc/agent/pkg/ring"
)

const defaultTokensPerNode = 256

type cluster struct {
	id         string
	membership membership.Membership
	logger     logging.Logger
	metrics    metrics.Metrics

	// The hash ring is used to determine which node is responsible for a given key.
	ring *ring.Ring[Node]

	// bearer token used to authenticate with other nodes
	authToken string
}

type Config struct {
	NodeId     string
	Membership membership.Membership
	Logger     logging.Logger
	Metrics    metrics.Metrics
	Debug      bool
	RpcAddr    string
	AuthToken  string
}

func New(config Config) (*cluster, error) {

	r, err := ring.New[Node](ring.Config{
		TokensPerNode: defaultTokensPerNode,
		Logger:        config.Logger,
	})
	if err != nil {
		return nil, err
	}

	c := &cluster{
		id:         config.NodeId,
		membership: config.Membership,
		logger:     config.Logger,
		metrics:    config.Metrics,
		ring:       r,
		authToken:  config.AuthToken,
	}

	go func() {
		joins := c.membership.SubscribeJoinEvents()
		leaves := c.membership.SubscribeLeaveEvents()
		for {
			select {
			case join := <-joins:
				err = r.AddNode(ring.Node[Node]{
					Id:   join.NodeId,
					Tags: Node{Id: join.NodeId, RpcAddr: join.RpcAddr},
				})
				if err != nil {
					c.logger.Error().Err(err).Str("nodeId", join.NodeId).Msg("unable to add node to ring")
				}
			case leave := <-leaves:
				err := r.RemoveNode(leave.NodeId)
				if err != nil {
					c.logger.Error().Err(err).Str("nodeId", leave.NodeId).Msg("unable to remove node from ring")
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

		prometheus.ClusterSize.Set(float64(len(members)))

	})

	// Do a forced sync every minute
	// I have observed that the channels can sometimes not be enough to keep the ring in sync
	repeat.Every(10*time.Second, func() {
		members, err := c.membership.Members()
		if err != nil {
			c.logger.Error().Err(err).Msg("failed to get members")
			return
		}
		existingMembers := c.ring.Members()
		c.logger.Debug().Int("want", len(members)).Int("have", len(existingMembers)).Msg("force syncing ring members")

		for _, existing := range existingMembers {
			found := false
			for _, m := range members {
				if m.NodeId == existing.Id {
					found = true
					break
				}
			}
			if !found {
				err := c.ring.RemoveNode(existing.Id)
				if err != nil {
					c.logger.Error().Err(err).Str("nodeId", existing.Id).Msg("unable to remove node from ring")
				}
			}
		}
		for _, m := range members {
			found := false
			for _, existing := range c.ring.Members() {
				if m.NodeId == existing.Id {
					found = true
					break
				}
			}
			if !found {
				err := c.ring.AddNode(ring.Node[Node]{
					Id:   m.NodeId,
					Tags: Node{Id: m.NodeId, RpcAddr: m.RpcAddr},
				})
				if err != nil {
					c.logger.Error().Err(err).Str("nodeId", m.NodeId).Msg("unable to add node to ring")
				}
			}
		}
	})

	return c, nil

}

func (c *cluster) NodeId() string {
	return c.id
}
func (c *cluster) Peers() []Node {
	members := c.ring.Members()
	nodes := []Node{}
	for _, m := range members {
		if m.Id == c.id {
			continue
		}
		nodes = append(nodes, m.Tags)
	}
	return nodes
}

func (c *cluster) Size() int {
	return len(c.ring.Members())
}

func (c *cluster) AuthToken() string {
	return c.authToken
}

func (c *cluster) FindNode(key string) (Node, error) {
	found, err := c.ring.FindNode(key)
	if err != nil {
		return Node{}, fmt.Errorf("failed to find node: %w", err)
	}

	return found.Tags, nil
}

func (c *cluster) Shutdown() error {
	c.logger.Info().Msg("shutting down cluster")

	// members, err := c.membership.Members()
	// if err != nil {
	// 	return fmt.Errorf("failed to get members: %w", err)

	// }

	err := c.membership.Leave()
	if err != nil {
		return fmt.Errorf("failed to leave membership: %w", err)
	}

	// ctx := context.Background()
	// wg := sync.WaitGroup{}
	// for _, m := range members {
	// 	wg.Add(1)
	// 	go func() {
	// 		defer wg.Done()

	// 		req := connect.NewRequest(&clusterv1.AnnounceStateChangeRequest{
	// 			NodeId: c.id,
	// 			State:  clusterv1.NodeState_NODE_STATE_LEAVING,
	// 		})
	// 		req.Header().Set("Authorization", c.authToken)

	// 		_, err := clusterv1connect.NewClusterServiceClient(http.DefaultClient, m.RpcAddr).AnnounceStateChange(ctx, req)
	// 		if err != nil {
	// 			c.logger.Error().Err(err).Str("peerId", m.NodeId).Msg("failed to announce state change")
	// 		}
	// 	}()
	// }
	// wg.Wait()
	return nil
}
