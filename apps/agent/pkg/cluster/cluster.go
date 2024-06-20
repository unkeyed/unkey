package cluster

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/membership"
)

type Cluster struct {
	id string

	membership *membership.Membership
	logger     logging.Logger

	nodesMu sync.RWMutex
	nodes   map[string]*Node
}

type Config struct {
	NodeId     string
	Membership *membership.Membership
	Logger     logging.Logger
	Debug      bool
}

func New(config Config) (*Cluster, error) {

	c := &Cluster{
		id:         config.NodeId,
		membership: config.Membership,
		logger:     config.Logger,
		nodesMu:    sync.RWMutex{},
		nodes:      make(map[string]*Node),
	}

	// err = c.ring.AddNode(ring.Node{
	// 	Id:              config.NodeId,
	// 	RpcAddr:         config.RpcAddr,
	// 	AvailabiltyZone: config.AvailabiltyZone,
	// })
	// if err != nil {
	// 	return nil, err
	// }

	go func() {
		joins := c.membership.SubscribeJoinEvents()
		leaves := c.membership.SubscribeLeaveEvents()
		broadcasts := c.membership.SubscribeGossipEvents()
		for {
			select {
			case broadcast := <-broadcasts:
				c.logger.Info().Str("event", broadcast.Event).Bytes("payload", broadcast.Payload).Msg("received gossip event")
			case join := <-joins:
				c.logger.Info().Str("node", join.Name).Msg("adding node to ring")
				tags := membership.Tags{}
				err := tags.Unmarshal(join.Tags)
				if err != nil {
					c.logger.Error().Err(err).Str("nodeId", join.Name).Msg("unable to unmarshal tags")
				}
				err = c.AddNode(Node{
					Id:     join.Name,
					Region: tags.Region,
				})
				if err != nil {
					c.logger.Error().Err(err).Str("nodeId", join.Name).Msg("unable to add node to cluster")

				}
			case leave := <-leaves:
				c.logger.Info().Str("node", leave.Name).Msg("removing node from cluster")
				err := c.RemoveNode(leave.Name)
				if err != nil {
					c.logger.Error().Err(err).Str("nodeId", leave.Name).Msg("unable to remove node from cluster")
				}

			}

		}
	}()

	if config.Debug {
		go func() {
			t := time.NewTicker(10 * time.Second)
			defer t.Stop()
			for range t.C {
				members := c.membership.Members()
				memberAddrs := make([]string, len(members))
				for i, member := range members {
					memberAddrs[i] = member.Name
				}
				c.logger.Info().Strs("members", memberAddrs).Int("clusterSize", len(members)).Str("nodeId", c.id).Send()
			}
		}()

	}

	go func() {
		t := time.NewTimer(time.Duration(rand.Intn(10)) * time.Second)
		defer t.Stop()
		for range t.C {
			err := c.membership.Broadcast(fmt.Sprintf("hello from %s", c.id), []byte("world"))
			if err != nil {
				c.logger.Error().Err(err).Msg("unable to broadcast")
			}
			t.Reset(time.Duration(rand.Intn(30)) * time.Second)
		}
	}()

	return c, nil

}

func (c *Cluster) AddNode(node Node) error {
	c.nodesMu.Lock()
	defer c.nodesMu.Unlock()

	c.logger.Info().Str("nodeId", node.Id).Msg("adding node to cluster")
	for _, n := range c.nodes {
		if n.Id == node.Id {
			return fmt.Errorf("node already exists: %s", node.Id)
		}

	}

	return nil
}

func (c *Cluster) RemoveNode(nodeId string) error {
	c.nodesMu.Lock()
	defer c.nodesMu.Unlock()

	delete(c.nodes, nodeId)

	return nil
}

func (c *Cluster) Join(addrs []string) error {
	c.logger.Info().Strs("addrs", addrs).Msg("joining cluster")
	return c.membership.Join(addrs...)
}

func (c *Cluster) Shutdown() error {
	c.logger.Info().Msg("shutting down cluster")
	return c.membership.Leave()
}
