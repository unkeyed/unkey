package membership

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/unkeyed/unkey/go/pkg/events"
	"github.com/unkeyed/unkey/go/pkg/logging"
)

type membership struct {
	mu          sync.Mutex
	started     bool
	logger      logging.Logger
	rdb         *redis.Client
	rpcAddr     string
	joinEvents  events.Topic[Member]
	leaveEvents events.Topic[Member]

	heartbeatLock   sync.Mutex
	membersSnapshot []Member

	nodeID string
}

type Config struct {
	RedisUrl string
	NodeID   string
	RpcAddr  string
	Logger   logging.Logger
}

func New(config Config) (Membership, error) {

	opts, err := redis.ParseURL(config.RedisUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}

	rdb := redis.NewClient(opts)

	_, err = rdb.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &membership{
		mu:              sync.Mutex{},
		started:         false,
		logger:          config.Logger.With(slog.String("pkg", "service discovery"), slog.String("type", "redis")),
		rdb:             rdb,
		rpcAddr:         config.RpcAddr,
		joinEvents:      events.NewTopic[Member](),
		leaveEvents:     events.NewTopic[Member](),
		nodeID:          config.NodeID,
		heartbeatLock:   sync.Mutex{},
		membersSnapshot: []Member{},
	}, nil

}

func (m *membership) Join(ctx context.Context, addrs ...string) (int, error) {

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started {
		return 0, fmt.Errorf("Membership already started")
	}
	m.started = true

	m.heartbeat(ctx)

	go func() {
		t := time.NewTicker(10 * time.Second)
		m.logger.Info(ctx, "heartbeating")
		defer t.Stop()
		for range t.C {
			m.heartbeat(ctx)
		}
	}()

	members, err := m.Members(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get members: %w", err)
	}
	return len(members), nil

}

func (m *membership) heartbeatRedisKey() string {
	return fmt.Sprintf("cluster::membership::nodes::%s", m.nodeID)
}

func (m *membership) heartbeat(ctx context.Context) {

	m.heartbeatLock.Lock()
	defer m.heartbeatLock.Unlock()
	b, err := json.Marshal(Member{
		ID:      m.nodeID,
		RpcAddr: m.rpcAddr,
	})
	if err != nil {
		m.logger.Error(ctx, "failed to marshal self", slog.String("error", err.Error()))
		return
	}
	err = m.rdb.Set(context.Background(), m.heartbeatRedisKey(), string(b), 15*time.Second).Err()
	if err != nil {
		m.logger.Error(ctx, "failed to set node", slog.String("error", err.Error()))
	}

	newMembers, err := m.Members(ctx)
	if err != nil {
		m.logger.Error(ctx, "failed to get members", slog.String("error", err.Error()))
		return
	}

	// Check for left nodes
	for _, oldMember := range m.membersSnapshot {
		found := false
		for _, newMember := range newMembers {
			if oldMember.ID == newMember.ID {
				found = true
				break
			}
		}
		if !found {
			m.leaveEvents.Emit(ctx, oldMember)
		}
	}

	// Check for new nodes
	for _, newMember := range newMembers {
		found := false
		for _, oldMember := range m.membersSnapshot {
			if oldMember.ID == newMember.ID {
				found = true
				break
			}
		}
		if !found {
			m.joinEvents.Emit(ctx, newMember)
		}

	}
	m.membersSnapshot = newMembers

}

func (m *membership) Leave(ctx context.Context) error {
	return m.rdb.Del(ctx, m.heartbeatRedisKey()).Err()
}

func (m *membership) Members(ctx context.Context) ([]Member, error) {
	pattern := m.heartbeatRedisKey()
	pattern = strings.ReplaceAll(pattern, m.nodeID, "*")

	keys, err := m.rdb.Keys(context.Background(), pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %w", err)
	}
	members := []Member{}
	values, err := m.rdb.MGet(context.Background(), keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}
	for _, val := range values {
		if val == nil {
			m.logger.Warn(ctx, "nil value found in members")
			continue
		}
		str, ok := val.(string)
		if !ok {
			m.logger.Error(ctx, "failed to cast value to string")
			return nil, fmt.Errorf("failed to cast value to string")
		}
		var member Member
		err = json.Unmarshal([]byte(str), &member)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal member: %w", err)
		}
		members = append(members, member)
	}

	return members, nil
}
func (m *membership) Addr() string {
	return m.rpcAddr
}
func (m *membership) SubscribeJoinEvents() <-chan Member {
	return m.joinEvents.Subscribe("cluster_join_events")
}

func (m *membership) SubscribeLeaveEvents() <-chan Member {
	return m.leaveEvents.Subscribe("cluster_leave_events")
}
