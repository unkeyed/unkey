package membership

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/unkeyed/unkey/go/pkg/events"
	"github.com/unkeyed/unkey/go/pkg/logging"
)

type fakeMembership struct {
	mu          sync.Mutex
	started     bool
	logger      logging.Logger
	rpcAddr     string
	joinEvents  events.Topic[Member]
	leaveEvents events.Topic[Member]

	members []Member

	nodeID string
}

type FakeConfig struct {
	NodeID  string
	RpcAddr string
	Logger  logging.Logger
}

func NewFake(config FakeConfig) (*fakeMembership, error) {

	return &fakeMembership{
		mu:          sync.Mutex{},
		logger:      config.Logger.With(slog.String("pkg", "service discovery"), slog.String("type", "fake")),
		rpcAddr:     config.RpcAddr,
		joinEvents:  events.NewTopic[Member](),
		leaveEvents: events.NewTopic[Member](),
		nodeID:      config.NodeID,
		members:     []Member{},
	}, nil

}

func (m *fakeMembership) AddMember(member Member) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, existing := range m.members {
		if existing.ID == member.ID {
			return
		}
	}
	m.members = append(m.members, member)
	m.joinEvents.Emit(context.Background(), member)
}

func (m *fakeMembership) RemoveMember(member Member) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, existing := range m.members {
		if existing.ID == member.ID {
			m.members[i] = m.members[len(m.members)-1]
			m.members = m.members[:len(m.members)-1]
			m.leaveEvents.Emit(context.Background(), member)

			return
		}
	}
}

func (m *fakeMembership) Join(ctx context.Context) (int, error) {

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started {
		return 0, fmt.Errorf("Membership already started")
	}
	m.started = true

	members, err := m.Members(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get members: %w", err)
	}
	return len(members), nil

}

func (m *fakeMembership) heartbeatRedisKey() string {
	return fmt.Sprintf("cluster::membership::nodes::%s", m.nodeID)
}

func (m *fakeMembership) Leave(ctx context.Context) error {
	return nil
}

func (m *fakeMembership) Members(ctx context.Context) ([]Member, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	membersCopy := make([]Member, len(m.members))
	copy(membersCopy, m.members)

	return membersCopy, nil
}
func (m *fakeMembership) Addr() string {
	return m.rpcAddr
}
func (m *fakeMembership) SubscribeJoinEvents() <-chan Member {
	return m.joinEvents.Subscribe("cluster_join_events")
}

func (m *fakeMembership) SubscribeLeaveEvents() <-chan Member {
	return m.leaveEvents.Subscribe("cluster_leave_events")
}
