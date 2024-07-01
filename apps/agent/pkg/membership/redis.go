package membership

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
)

type membership struct {
	sync.Mutex
	started     bool
	logger      logging.Logger
	rdb         *redis.Client
	rpcAddr     string
	joinEvents  events.Topic[Member]
	leaveEvents events.Topic[Member]

	heartbeatLock   sync.Mutex
	membersSnapshot []Member

	nodeId string
}

type Config struct {
	RedisUrl string
	NodeId   string
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
		started:         false,
		logger:          config.Logger,
		rdb:             rdb,
		rpcAddr:         config.RpcAddr,
		joinEvents:      events.NewTopic[Member](),
		leaveEvents:     events.NewTopic[Member](),
		nodeId:          config.NodeId,
		heartbeatLock:   sync.Mutex{},
		membersSnapshot: []Member{},
	}, nil

}

func (m *membership) Join(addrs ...string) (int, error) {

	m.Lock()
	defer m.Unlock()
	if m.started {
		return 0, fmt.Errorf("Membership already started")
	}
	m.started = true
	m.logger.Info().Msg("Initilizing redis membership")

	m.heartbeat()

	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for range t.C {
			m.heartbeat()
		}
	}()

	members, err := m.Members()
	if err != nil {
		return 0, fmt.Errorf("failed to get members: %w", err)
	}
	return len(members), nil

}

func (m *membership) heartbeatRedisKey() string {
	return fmt.Sprintf("cluster::membership::nodes::%s", m.nodeId)
}

func (m *membership) heartbeat() {
	m.heartbeatLock.Lock()
	defer m.heartbeatLock.Unlock()
	b, err := json.Marshal(Member{
		Id:      m.nodeId,
		RpcAddr: m.rpcAddr,
	})
	if err != nil {
		m.logger.Err(err).Msg("failed to marshal self")
		return
	}
	err = m.rdb.Set(context.Background(), m.heartbeatRedisKey(), string(b), 15*time.Second).Err()
	if err != nil {
		m.logger.Error().Err(err).Msg("failed to set node")
	}

	newMembers, err := m.Members()
	if err != nil {
		m.logger.Error().Err(err).Msg("failed to get members")
		return
	}

	// Check for left nodes
	for _, oldMember := range m.membersSnapshot {
		found := false
		for _, newMember := range newMembers {
			if oldMember.Id == newMember.Id {
				found = true
				break
			}
		}
		if !found {
			m.leaveEvents.Emit(oldMember)
		}
	}

	// Check for new nodes
	for _, newMember := range newMembers {
		found := false
		for _, oldMember := range m.membersSnapshot {
			if oldMember.Id == newMember.Id {
				found = true
				break
			}
		}
		if !found {
			m.joinEvents.Emit(newMember)
		}

	}
	m.membersSnapshot = newMembers

}

func (m *membership) Leave() error {
	return m.rdb.Del(context.Background(), m.heartbeatRedisKey()).Err()
}

func (m *membership) Members() ([]Member, error) {
	pattern := m.heartbeatRedisKey()
	pattern = strings.ReplaceAll(pattern, m.nodeId, "*")

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
			m.logger.Warn().Msg("nil value found in members")
			continue
		}
		str, ok := val.(string)
		if !ok {
			m.logger.Error().Interface("val", val).Msg("failed to cast value to string")
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
	return m.joinEvents.Subscribe()
}

func (m *membership) SubscribeLeaveEvents() <-chan Member {
	return m.leaveEvents.Subscribe()
}
