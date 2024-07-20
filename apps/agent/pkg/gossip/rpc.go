package gossip

import (
	"bytes"
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	gossipv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/gossip/v1"
	"github.com/unkeyed/unkey/apps/agent/gen/proto/gossip/v1/gossipv1connect"
	"google.golang.org/protobuf/proto"
)

func (c *cluster) join(ctx context.Context, req *gossipv1.JoinRequest) (*gossipv1.JoinResponse, error) {
	c.logger.Info().Str("peerId", req.Self.NodeId).Msg("peer is asking to join")

	newMember := Member{
		NodeId:  req.Self.NodeId,
		RpcAddr: req.Self.RpcAddr,
	}

	c.Lock()
	defer c.Unlock()

	existing, ok := c.members[req.Self.NodeId]
	if !ok {
		c.memberJoinTopic.Emit(ctx, newMember)
		c.members[req.Self.NodeId] = req.Self
	} else {
		e, err := proto.Marshal(existing)
		if err != nil {
			return nil, fault.Wrap(err, fmsg.With("failed to marshal existing member"))
		}
		j, err := proto.Marshal(req.Self)
		if err != nil {
			return nil, fault.Wrap(err, fmsg.With("failed to marshal new member"))
		}
		if !bytes.Equal(e, j) {
			c.memberUpdateTopic.Emit(ctx, newMember)
			c.members[req.Self.NodeId] = req.Self
		}

	}

	members := []*gossipv1.Member{}
	for _, m := range c.members {
		members = append(members, m)
	}

	return &gossipv1.JoinResponse{
		Members: members,
	}, nil
}

func (c *cluster) leave(ctx context.Context, req *gossipv1.LeaveRequest) (*gossipv1.LeaveResponse, error) {
	c.Lock()
	delete(c.members, req.Self.NodeId)
	c.Unlock()
	c.memberLeaveTopic.Emit(ctx, Member{
		NodeId:  req.Self.NodeId,
		RpcAddr: req.Self.RpcAddr,
	})

	return &gossipv1.LeaveResponse{}, nil
}

func (c *cluster) ping(
	ctx context.Context,
	req *gossipv1.PingRequest,
) (*gossipv1.PingResponse, error) {

	return &gossipv1.PingResponse{
		State: gossipv1.State_State_ALIVE,
	}, nil
}

func (c *cluster) indirectPing(
	ctx context.Context,
	req *gossipv1.IndirectPingRequest,
) (*gossipv1.IndirectPingResponse, error) {
	peer := gossipv1connect.NewGossipServiceClient(http.DefaultClient, req.RpcAddr)

	pong, err := peer.Ping(ctx, connect.NewRequest(&gossipv1.PingRequest{}))

	switch pong.Msg.State {
	case gossipv1.State_State_ALIVE:

	default:
		c.removeMemberFromState(ctx, req.NodeId)
	}

	if err != nil {
		return nil, fault.Wrap(err, fmsg.With("unable to ping peer"))
	}

	return &gossipv1.IndirectPingResponse{
		State: pong.Msg.State,
	}, nil
}

func (c *cluster) syncMembers(
	ctx context.Context,
	req *gossipv1.SyncMembersRequest,
) (*gossipv1.SyncMembersResponse, error) {
	c.Lock()
	defer c.Unlock()

	union := map[string]*gossipv1.Member{}
	// Add all existing members to the union
	for _, m := range c.members {
		union[m.NodeId] = m
	}

	// Add all new members to the union
	for _, m := range req.Members {
		_, ok := union[m.NodeId]
		if !ok {
			union[m.NodeId] = m
		} else if m.State == gossipv1.State_State_ALIVE {
			union[m.NodeId] = m
		}
	}

	arr := []*gossipv1.Member{}
	for _, m := range union {
		arr = append(arr, m)
	}
	c.members = union

	return &gossipv1.SyncMembersResponse{
		Members: arr,
	}, nil

}
