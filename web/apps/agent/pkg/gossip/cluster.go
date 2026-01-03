package gossip

import (
	"context"
	"net/http"
	"sync"
	"time"

	"math/rand"

	"connectrpc.com/connect"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	gossipv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/gossip/v1"
	"github.com/unkeyed/unkey/apps/agent/gen/proto/gossip/v1/gossipv1connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

// ensure cluster implements Cluster
var _ Cluster = &cluster{}

type cluster struct {
	sync.RWMutex
	logger logging.Logger

	self *gossipv1.Member

	// all members of the cluster, including self
	members map[string]*gossipv1.Member

	config Config

	memberJoinTopic   events.Topic[Member]
	memberUpdateTopic events.Topic[Member]
	memberLeaveTopic  events.Topic[Member]

	shutdown events.Topic[bool]
}

type Config struct {
	Logger logging.Logger

	NodeId  string
	RpcAddr string

	// How frequently to gossip with other members
	GossipInterval time.Duration

	// Timeout for gossip requests, if a member doesn't respond within this time, it is considered a
	// suspect
	GossipTimeout time.Duration

	// Each interval, a member will gossip to this many other members
	GossipFactor int
}

func (c Config) withDefaults() Config {
	if c.GossipInterval == 0 {
		c.GossipInterval = time.Second
	}
	if c.GossipTimeout == 0 {
		c.GossipTimeout = time.Second
	}
	if c.GossipFactor == 0 {
		c.GossipFactor = 3
	}

	return c

}

func New(config Config) (*cluster, error) {

	self := &gossipv1.Member{
		NodeId:  config.NodeId,
		RpcAddr: config.RpcAddr,
		State:   gossipv1.State_State_ALIVE,
	}

	c := &cluster{
		logger:            config.Logger,
		self:              self,
		members:           map[string]*gossipv1.Member{},
		config:            config.withDefaults(),
		shutdown:          events.NewTopic[bool](),
		memberJoinTopic:   events.NewTopic[Member](),
		memberUpdateTopic: events.NewTopic[Member](),
		memberLeaveTopic:  events.NewTopic[Member](),
	}

	c.members[self.NodeId] = self

	return c, nil
}

// Run starts the cluster's gossip loop and other background tasks
//
// Stops automatic when a message from the shutdown topic is received
func (c *cluster) run() {
	stop := c.shutdown.Subscribe("cluster shutdown")
	t := time.NewTicker(c.config.GossipInterval)

	for {
		select {
		case <-stop:
			t.Stop()
			return
		case <-t.C:
			err := c.gossip(context.Background())
			if err != nil {
				c.logger.Warn().Err(err).Msg("failed to gossip")
			}

		}
	}
}

func (c *cluster) RpcAddr() string {
	return c.self.RpcAddr
}

func (c *cluster) Members() map[string]Member {
	c.RLock()
	defer c.RUnlock()

	members := map[string]Member{}
	for k, v := range c.members {
		members[k] = Member{
			NodeId:  v.NodeId,
			RpcAddr: v.RpcAddr,
		}
	}

	return members
}

func (c *cluster) Join(ctx context.Context, rpcAddrs ...string) error {
	c.logger.Info().Strs("rpcAddrs", rpcAddrs).Msg("attempting to join cluster")

	c.Lock()
	defer c.Unlock()

	successfullyExchanged := 0
	errors := []error{}

	for _, rpcAddr := range rpcAddrs {
		if rpcAddr == c.self.RpcAddr {
			// Skip talking to ourselves
			continue
		}

		client := gossipv1connect.NewGossipServiceClient(http.DefaultClient, rpcAddr)

		var resp *connect.Response[gossipv1.JoinResponse]
		err := util.Retry(func() error {
			var joinErr error
			resp, joinErr = client.Join(ctx, connect.NewRequest(&gossipv1.JoinRequest{
				Self: &gossipv1.Member{
					NodeId:  c.self.NodeId,
					RpcAddr: c.self.RpcAddr,
					State:   gossipv1.State_State_ALIVE,
				},
			}))
			if joinErr != nil {
				c.logger.Warn().Err(joinErr).Str("rpcAddr", rpcAddr).Msg("error joining cluster")

				return joinErr
			}
			return nil
		}, 5, func(n int) time.Duration {
			return time.Duration(n) * time.Second
		})

		if err != nil {
			errors = append(errors, err)
			continue
		}

		for _, m := range resp.Msg.Members {
			c.members[m.NodeId] = m
			c.memberJoinTopic.Emit(ctx, Member{
				NodeId:  m.NodeId,
				RpcAddr: m.RpcAddr,
			})
		}

		successfullyExchanged++
	}
	if (float64(successfullyExchanged) / float64(len(rpcAddrs))) >= 0.5 {
		// If more than half of the members successfully exchanged, consider the join successful
		return nil
	}

	if len(errors) > 0 {
		return fault.Wrap(errors[0], fmsg.With("failed to join cluster"))
	}

	// After joining the cluster, start the gossip loop
	go c.run()
	return nil
}

func (c *cluster) Shutdown(ctx context.Context) error {

	c.shutdown.Emit(ctx, true)

	c.Lock()
	defer c.Unlock()

	errors := []error{}

	for _, member := range c.members {
		if member.NodeId == c.self.NodeId {
			// Skip talking to ourselves
			continue
		}

		client := gossipv1connect.NewGossipServiceClient(http.DefaultClient, member.RpcAddr)

		err := util.Retry(func() error {
			_, leaveError := client.Leave(ctx, connect.NewRequest(&gossipv1.LeaveRequest{
				Self: &gossipv1.Member{
					NodeId:  c.self.NodeId,
					RpcAddr: c.self.RpcAddr,
					State:   gossipv1.State_State_LEFT,
				},
			}))
			if leaveError != nil {
				c.logger.Warn().Err(leaveError).Str("rpcAddr", member.RpcAddr).Msg("error leaving cluster")

				return leaveError
			}
			return nil
		}, 5, func(n int) time.Duration {
			return time.Duration(n) * time.Second
		})

		if err != nil {
			errors = append(errors, err)
			continue
		}

	}
	if len(errors) > 0 {
		return fault.Wrap(errors[0], fmsg.With("failed to leave cluster"))

	}

	return nil
}

func (c *cluster) SubscribeJoinEvents(callerName string) <-chan Member {
	return c.memberJoinTopic.Subscribe(callerName)
}

func (c *cluster) SubscribeUpdateEvents(callerName string) <-chan Member {
	return c.memberUpdateTopic.Subscribe(callerName)
}

func (c *cluster) SubscribeLeaveEvents(callerName string) <-chan Member {
	return c.memberLeaveTopic.Subscribe(callerName)
}

func (c *cluster) randomPeers(n int, withoutNodeIds ...string) ([]*gossipv1.Member, error) {
	c.RLock()
	defer c.RUnlock()

	peerIds := make([]string, 0, len(c.members))
	for id := range c.members {
		if id == c.self.NodeId {
			continue
		}
		peerIds = append(peerIds, id)
	}

	peers := []*gossipv1.Member{}
	for len(peers) < n {
		peer := c.members[peerIds[rand.Intn(len(peerIds))]]
		if len(withoutNodeIds) > 0 {
			for _, withoutNodeId := range withoutNodeIds {
				if peer.NodeId == withoutNodeId {
					continue
				}
			}

		}

		peers = append(peers, peer)
	}

	return peers, nil
}

func (c *cluster) addMemberToState(ctx context.Context, member *gossipv1.Member) {
	c.Lock()
	defer c.Unlock()

	_, ok := c.members[member.NodeId]

	c.members[member.NodeId] = member

	if !ok {
		c.memberJoinTopic.Emit(ctx, Member{
			NodeId:  member.NodeId,
			RpcAddr: member.RpcAddr,
		})
	}
}

func (c *cluster) removeMemberFromState(ctx context.Context, nodeId string) {
	c.Lock()
	defer c.Unlock()

	member, ok := c.members[nodeId]
	if !ok {
		return
	}

	delete(c.members, member.NodeId)
	c.memberLeaveTopic.Emit(ctx, Member{
		NodeId:  member.NodeId,
		RpcAddr: member.RpcAddr,
	})
}

func (c *cluster) gossip(ctx context.Context) error {

	peers, err := c.randomPeers(c.config.GossipFactor)
	if err != nil {
		return fault.Wrap(err, fmsg.With("failed to find peers to gossip with"))
	}

	for _, peer := range peers {
		c.logger.Debug().Str("peerId", peer.NodeId).Msg("gossiping about membership with peer")
		client := gossipv1connect.NewGossipServiceClient(http.DefaultClient, peer.RpcAddr)
		ctxWithTimeout, cancel := context.WithTimeout(ctx, c.config.GossipTimeout)
		defer cancel()
		res, err := client.Ping(ctxWithTimeout, connect.NewRequest(&gossipv1.PingRequest{}))

		if err == nil {
			switch res.Msg.State {
			case gossipv1.State_State_ALIVE:
				c.logger.Debug().Str("peerId", peer.NodeId).Msg("peer is alive")
				continue
			case gossipv1.State_State_LEFT:
				c.logger.Debug().Str("peerId", peer.NodeId).Msg("peer has left")
				c.removeMemberFromState(ctx, peer.NodeId)
				continue
			default:
				c.logger.Debug().Str("peerId", peer.NodeId).Msg("peer is not alive")
			}
		}

		// Peer was not alive, let's check via indirect gossip

		indirectPeers, err := c.randomPeers(c.config.GossipFactor, peer.NodeId)
		if err != nil {
			return fault.Wrap(err, fmsg.With("failed to find indirect peers to gossip with"))
		}

		for _, indirectPeer := range indirectPeers {
			c.logger.Debug().Str("peerId", indirectPeer.NodeId).Msg("gossiping about membership with indirect peer")
			client := gossipv1connect.NewGossipServiceClient(http.DefaultClient, indirectPeer.RpcAddr)
			ctxWithTimeout, cancel := context.WithTimeout(ctx, c.config.GossipTimeout)
			defer cancel()
			res, err := client.IndirectPing(ctxWithTimeout, connect.NewRequest(&gossipv1.IndirectPingRequest{
				NodeId:  peer.NodeId,
				RpcAddr: peer.RpcAddr,
			}))
			if err != nil {
				return fault.Wrap(err, fmsg.With("failed to gossip with indirect peer"))
			}
			switch res.Msg.State {
			case gossipv1.State_State_ALIVE:
				c.logger.Debug().Str("peerId", indirectPeer.NodeId).Msg("indirect peer is alive")
			default:
				c.logger.Debug().Str("peerId", indirectPeer.NodeId).Msg("indirect peer is not alive")
				c.removeMemberFromState(ctx, peer.NodeId)
			}

		}
	}

	// // sync with one random node

	// peers, err = c.randomPeers(1)
	// if err != nil {
	// 	return fault.Wrap(err, fmsg.With("failed to find peers to sync with"))
	// }
	// client := gossipv1connect.NewGossipServiceClient(http.DefaultClient, peers[0].RpcAddr)
	// ctxWithTimeout, cancel := context.WithTimeout(ctx, c.config.GossipTimeout)
	// defer cancel()

	// arr := []*gossipv1.Member{}
	// c.RLock()
	// for _, m := range c.members {
	// 	arr = append(arr, m)
	// }
	// c.RUnlock()
	// res, err := client.SyncMembers(ctxWithTimeout, connect.NewRequest(&gossipv1.SyncMembersRequest{
	// 	Members: arr,
	// }))
	// if err != nil {
	// 	return fault.Wrap(err, fmsg.With("failed to sync with peer"))
	// }

	// c.Lock()
	// defer c.Unlock()
	// for _, m := range res.Msg.Members {
	// 	_, ok := c.members[m.NodeId]
	// 	if !ok {
	// 		c.members[m.NodeId] = m
	// 	} else if m.State == gossipv1.State_State_ALIVE {
	// 		c.members[m.NodeId] = m
	// 	}
	// }

	return nil

}
