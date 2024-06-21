package cluster

import (
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/membership"
	"github.com/unkeyed/unkey/apps/agent/pkg/ring"
)

type Cluster struct {
	id         string
	membership membership.Membership
	logger     logging.Logger

	// The hash ring is used to determine which node is responsible for a given key.
	ring *ring.Ring

	shutdownCh chan struct{}
}

type Config struct {
	NodeId     string
	Membership membership.Membership
	Logger     logging.Logger
	Debug      bool
	RpcAddr    string
}

func New(config Config) (*Cluster, error) {

	r, err := ring.New(ring.Config{
		TokensPerNode: 256,
		Logger:        config.Logger,
	})
	if err != nil {
		return nil, err
	}

	c := &Cluster{
		id:         config.NodeId,
		membership: config.Membership,
		logger:     config.Logger,
		ring:       r,
		shutdownCh: make(chan struct{}),
	}

	go func() {
		joins := c.membership.SubscribeJoinEvents()
		leaves := c.membership.SubscribeLeaveEvents()
		for {
			select {
			case join := <-joins:
				c.logger.Info().Str("node", join.Id).Msg("adding node to ring")
				err = r.AddNode(ring.Node{
					Id:      join.Id,
					RpcAddr: join.RpcAddr,
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

	if config.Debug {
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
					c.logger.Info().Strs("members", memberAddrs).Int("clusterSize", len(members)).Str("nodeId", c.id).Send()
				}
			}
		}()

	}

	if config.Debug {
		go func() {
			t := time.NewTicker(10 * time.Second)
			defer t.Stop()
			for {
				select {
				case <-c.shutdownCh:
					return
				case <-t.C:
					err := c.heartbeat()
					if err != nil {
						c.logger.Error().Err(err).Msg("failed to heartbeat")
					}
				}
			}
		}()

	}

	return c, nil

}

func (c *Cluster) heartbeat() error {
	members, err := c.membership.Members()
	if err != nil {
		return fmt.Errorf("failed to get members: %w", err)
	}
	for _, member := range members {
		if member.Id == c.id {
			continue
		}
		req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/v1/liveness", member.RpcAddr), nil)

		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to do request: %w", err)
		}
		defer res.Body.Close()
		c.logger.Info().Int("status", res.StatusCode).Str("peer", member.Id).Msg("heartbeat")
	}
	return nil

}

func (c *Cluster) FindNode(key string) (*ring.Node, error) {
	return c.ring.FindNode(key)
}
func (c *Cluster) Join(addrs []string) (clusterSize int, err error) {
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

func (c *Cluster) Shutdown() error {
	close(c.shutdownCh)
	return c.membership.Leave()
}
