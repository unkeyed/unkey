package membership

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	clusterv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/cluster/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/repeat"
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

	syncTtl       time.Duration
	syncFrequency time.Duration

	nodeId string
}

type Config struct {
	RedisUrl string
	NodeId   string
	RpcAddr  string
	Logger   logging.Logger

	// How frequently to heartbeat to redis to indicate that this node is still alive as well as to
	// download the list of other nodes.
	SyncFrequency time.Duration

	// The time to live for the heartbeat key in redis. If this node fails to heartbeat within this
	// time, it will be considered dead.
	//
	// Set this to roughly 2x the sync frequency or at least 5s longer than the sync frequency.
	SyncTtl time.Duration
}

func New(config Config) (Membership, error) {

	opts, err := redis.ParseURL(config.RedisUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}
	opts.MaxRetryBackoff = time.Second
	opts.MaxRetries = 5

	rdb := redis.NewClient(opts)

	_, err = rdb.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	if config.SyncTtl == 0 {
		config.SyncTtl = 15 * time.Second
	}
	if config.SyncFrequency == 0 {
		config.SyncFrequency = 10 * time.Second
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

		syncTtl:       config.SyncTtl,
		syncFrequency: config.SyncFrequency,
	}, nil
}

func (m *membership) NodeId() string {
	return m.nodeId
}

func (m *membership) Join(addrs ...string) (int, error) {

	m.Lock()
	defer m.Unlock()
	if m.started {
		return 0, fmt.Errorf("Membership already started")
	}
	m.started = true
	m.logger.Info().Msg("Initilizing redis membership")

	err := m.Sync()
	if err != nil {
		return 0, fmt.Errorf("failed to sync: %w", err)
	}

	repeat.Every(m.syncFrequency, func() {
		err := m.Sync()
		if err != nil {
			m.logger.Error().Err(err).Msg("failed to sync")
		}
	})

	members, err := m.Members()
	if err != nil {
		return 0, fmt.Errorf("failed to get members: %w", err)
	}
	return len(members), nil

}

func (m *membership) heartbeatRedisKey() string {
	return fmt.Sprintf("cluster::membership::nodes::%s", m.nodeId)
}

func (m *membership) Sync() error {
	m.heartbeatLock.Lock()
	defer m.heartbeatLock.Unlock()
	b, err := json.Marshal(Member{
		Id:      m.nodeId,
		RpcAddr: m.rpcAddr,
		State:   clusterv1.NodeState_NODE_STATE_ACTIVE.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal self: %w", err)
	}
	err = m.rdb.Set(context.Background(), m.heartbeatRedisKey(), string(b), m.syncFrequency).Err()
	if err != nil {
		m.logger.Error().Err(err).Msg("failed to set node")
	}

	newMembers, err := m.Members()
	if err != nil {
		return fmt.Errorf("failed to get members: %w", err)
	}

	// Check for left nodes
	for _, oldMember := range m.membersSnapshot {
		found := false
		for _, newMember := range newMembers {
			if oldMember.Id == newMember.Id && newMember.State == clusterv1.NodeState_NODE_STATE_ACTIVE.String() {
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
		if !found && newMember.State == clusterv1.NodeState_NODE_STATE_ACTIVE.String() {
			m.joinEvents.Emit(newMember)
		}

	}
	m.membersSnapshot = newMembers
	return nil

}

func (m *membership) Leave() error {

	b, err := json.Marshal(Member{
		Id:      m.nodeId,
		RpcAddr: m.rpcAddr,
		State:   clusterv1.NodeState_NODE_STATE_LEAVING.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal self: %w", err)
	}

	err = m.rdb.Set(context.Background(), m.heartbeatRedisKey(), string(b), m.syncFrequency).Err()
	if err != nil {
		return fmt.Errorf("failed to set node: %w", err)
	}
	return nil
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
