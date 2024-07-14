package gossip

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	gossipv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/gossip/v1"
	"github.com/unkeyed/unkey/apps/agent/gen/proto/gossip/v1/gossipv1connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/auth"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
)

type cluster struct {
	sync.RWMutex
	logger logging.Logger

	self *gossipv1.Member

	// all members of the cluster, including self
	members map[string]*gossipv1.Member

	config Config

	shutdown events.Topic[bool]

	gossipv1connect.UnimplementedGossipServiceHandler
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

func New(config Config) *cluster {

	self := &gossipv1.Member{
		NodeId:  config.NodeId,
		RpcAddr: config.RpcAddr,
	}

	c := &cluster{
		logger:   config.Logger,
		self:     self,
		members:  map[string]*gossipv1.Member{self.NodeId: self},
		config:   config,
		shutdown: events.NewTopic[bool](),
	}

	return c
}

func (c *cluster) CreateHandler() (string, http.Handler, error) {
	otelInterceptor, err := otelconnect.NewInterceptor()
	if err != nil {
		return "", nil, err
	}

	path, handler := gossipv1connect.NewGossipServiceHandler(c, connect.WithInterceptors(otelInterceptor))
	return path, handler, nil

}

func (c *cluster) Run() {

	go func() {
		stop := c.shutdown.Subscribe()
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
	}()

}

func (c *cluster) gossip(ctx context.Context) error {
	c.RLock()

	peerIds := make([]string, 0, len(c.members))
	for id := range c.members {
		peerIds = append(peerIds, id)
	}

	peers := map[string]*gossipv1.Member{}
	for len(peers) < c.config.GossipFactor {
		peer := c.members[peerIds[rand.Intn(len(peerIds))]]
		if peer.NodeId != c.self.NodeId && peers[peer.NodeId] == nil {
			peers[peer.NodeId] = peer
		}
	}
	c.RUnlock()

	for _, peer := range peers {
		client := gossipv1connect.NewGossipServiceClient(http.DefaultClient, peer.RpcAddr)
		ctxWithTimeout, cancel := context.WithTimeout(ctx, c.config.GossipTimeout)
		defer cancel()
		_, err := client.Ping(ctxWithTimeout, connect.NewRequest(&gossipv1.PingRequest{}))
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				c.logger.Warn().Str("peer", peer.NodeId).Msg("peer did not respond in time")
				c.Lock()
				member, ok := c.members[peer.NodeId]
				if ok {
					member.State = gossipv1.State_State_SUSPECT
				}

				continue
			}
			c.logger.Warn().Err(err).Str("peer", peer.NodeId).Msg("failed to gossip with peer")
		}
	}

	return nil

}

func (c *cluster) Ping(
	ctx context.Context,
	req *connect.Request[gossipv1.PingRequest],
) (*connect.Response[gossipv1.PingResponse], error) {
	authorization := req.Header().Get("Authorization")
	err := auth.Authorize(ctx, "TODO:", authorization)
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to authorize request")
		return nil, err
	}

	return connect.NewResponse(&gossipv1.PingResponse{}), nil
}

func (c *cluster) IndirectPing(
	ctx context.Context,
	req *connect.Request[gossipv1.IndirectPingRequest],
) (*connect.Response[gossipv1.IndirectPingResponse], error) {
	authorization := req.Header().Get("Authorization")
	err := auth.Authorize(ctx, "TODO:", authorization)
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to authorize request")
		return nil, err

	}
	peer := gossipv1connect.NewGossipServiceClient(http.DefaultClient, req.Msg.RpcAddr)

	pong, err := peer.Ping(ctx, connect.NewRequest(&gossipv1.PingRequest{}))

	if err != nil {
		return nil, fault.Wrap(err, fmsg.With("unable to ping peer"))
	}

	return connect.NewResponse(&gossipv1.IndirectPingResponse{
		State: pong.Msg.State,
	}), nil
}

func (c *cluster) SyncMembers(
	ctx context.Context,
	req *connect.Request[gossipv1.SyncMembersRequest],
) (*connect.Response[gossipv1.SyncMembersResponse], error) {
	authorization := req.Header().Get("Authorization")
	err := auth.Authorize(ctx, "TODO:", authorization)
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to authorize request")
		return nil, err

	}

	return nil, fault.New("not implemented")

}
