package membership

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/serf/serf"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

type Config struct {
	NodeId   string
	SerfAddr string
	RpcAddr  string
	Logger   logging.Logger
	Region   string
}

// Reexport to hide serf dependency
type Member = serf.Member

type GossipEvent struct {
	Event   string
	Payload []byte
}

type Membership struct {
	sync.Mutex
	nodeId   string
	serfAddr string
	rpcAddr  string

	tags         Tags
	joinEvents   events.Topic[Member]
	leaveEvents  events.Topic[Member]
	gossipEvents events.Topic[GossipEvent]
	serf         *serf.Serf
	events       chan serf.Event
	logger       logging.Logger
	started      bool
}

func New(config Config) (*Membership, error) {

	m := &Membership{
		nodeId:   config.NodeId,
		serfAddr: config.SerfAddr,
		rpcAddr:  config.RpcAddr,
		tags: Tags{
			NodeId:   config.NodeId,
			SerfAddr: config.SerfAddr,
			RpcAddr:  config.RpcAddr,
			Region:   config.Region,
		},
		logger:       config.Logger.With().Str("node", config.NodeId).Str("serfAddr", config.SerfAddr).Str("rpcAddr", config.RpcAddr).Logger(),
		joinEvents:   events.NewTopic[Member](),
		leaveEvents:  events.NewTopic[Member](),
		gossipEvents: events.NewTopic[GossipEvent](),
	}
	return m, nil
}

func (m *Membership) SerfAddr() string {
	return m.serfAddr
}
func (m *Membership) SubscribeJoinEvents() <-chan Member {
	return m.joinEvents.Subscribe()
}

func (m *Membership) SubscribeLeaveEvents() <-chan Member {
	return m.leaveEvents.Subscribe()
}

func (m *Membership) SubscribeGossipEvents() <-chan GossipEvent {
	return m.gossipEvents.Subscribe()
}

func (m *Membership) Shutdown() error {
	m.Lock()
	defer m.Unlock()
	if !m.started {
		return fmt.Errorf("Membership was never started")
	}
	err := m.serf.Leave()
	if err != nil {
		return fmt.Errorf("Failed to leave serf: %w", err)
	}
	return m.serf.Shutdown()
}

func (m *Membership) Join(joinAddrs ...string) error {
	m.Lock()
	defer m.Unlock()
	if m.started {
		return fmt.Errorf("Membership already started")
	}
	m.started = true
	m.logger.Info().Msg("Initilizing serf")

	addr, err := net.ResolveTCPAddr("tcp", m.serfAddr)
	if err != nil {
		return fmt.Errorf("Failed to resolve serf address: %s", err)
	}
	config := serf.DefaultConfig()
	config.MemberlistConfig.BindAddr = addr.IP.String()
	config.MemberlistConfig.BindPort = addr.Port

	m.events = make(chan serf.Event)
	config.EventCh = m.events
	config.Tags, err = m.tags.Marshal()
	if err != nil {
		return fmt.Errorf("Failed to convert tags to map: %w", err)
	}
	config.NodeName = m.nodeId

	m.serf, err = serf.Create(config)
	if err != nil {
		return fmt.Errorf("Failed to create serf: %w", err)
	}

	m.logger.Info().Msg("Config is initialized")

	go m.eventHandler()
	if len(joinAddrs) > 0 {
		m.logger.Info().Strs("addrs", joinAddrs).Msg("Joining serf cluster")
		err := util.Retry(
			func() error {
				_, joinErr := m.serf.Join(joinAddrs, true)
				return joinErr
			},
			10,
			func(n int) time.Duration { return time.Duration(n) * time.Second },
		)
		if err != nil {
			return fmt.Errorf("Failed to join: %w", err)
		}
	}

	return nil
}

func (m *Membership) Broadcast(eventType string, payload []byte) error {
	return m.serf.UserEvent(eventType, payload, true)
}
func (m *Membership) eventHandler() {
	for e := range m.events {
		m.logger.Info().Str("type", e.EventType().String()).Msg("Event")
		switch e.EventType() {
		case serf.EventMemberJoin:
			for _, member := range e.(serf.MemberEvent).Members {
				if m.isLocal(member) {
					continue
				}
				m.joinEvents.Emit(member)
			}
		case serf.EventMemberLeave, serf.EventMemberFailed:
			for _, member := range e.(serf.MemberEvent).Members {
				if m.isLocal(member) {
					continue
				}
				m.leaveEvents.Emit(member)
			}
		case serf.EventUser:
			m.gossipEvents.Emit(GossipEvent{
				Event:   e.(serf.UserEvent).Name,
				Payload: e.(serf.UserEvent).Payload,
			})
		}
	}
}

func (m *Membership) isLocal(member serf.Member) bool {
	return member.Name == m.nodeId
}

func (m *Membership) Members() []serf.Member {
	members := make([]serf.Member, 0)
	for _, m := range m.serf.Members() {
		if m.Status == serf.StatusAlive {
			members = append(members, m)
		}
	}
	return members
}
func (m *Membership) Leave() error {
	return m.serf.Leave()
}
