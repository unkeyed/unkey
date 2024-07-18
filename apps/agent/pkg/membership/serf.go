package membership

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/hashicorp/serf/serf"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

type Config struct {
	NodeId   string
	SerfAddr string
	Logger   logging.Logger
	RpcAddr  string
}

type gossipEvent struct {
	event   string
	payload []byte
}

type membership struct {
	sync.Mutex
	serfAddr string

	self         Member
	joinEvents   events.Topic[Member]
	leaveEvents  events.Topic[Member]
	gossipEvents events.Topic[gossipEvent]
	serf         *serf.Serf
	events       chan serf.Event
	logger       logging.Logger
	started      bool
}

func New(config Config) (Membership, error) {
	m := &membership{
		serfAddr: config.SerfAddr,
		self: Member{
			NodeId:   config.NodeId,
			SerfAddr: config.SerfAddr,
			RpcAddr:  config.RpcAddr,
			State:    "alive",
		},
		logger:       config.Logger.With().Str("node", config.NodeId).Str("SerfAddr", config.SerfAddr).Logger(),
		joinEvents:   events.NewTopic[Member](),
		leaveEvents:  events.NewTopic[Member](),
		gossipEvents: events.NewTopic[gossipEvent](),
	}

	return m, nil
}
func (m *membership) NodeId() string {
	return m.self.NodeId
}

func (m *membership) SerfAddr() string {
	return m.serfAddr
}
func (m *membership) SubscribeJoinEvents() <-chan Member {
	return m.joinEvents.Subscribe("serfJoinEvents")
}

func (m *membership) SubscribeLeaveEvents() <-chan Member {
	return m.leaveEvents.Subscribe("serfLeaveEvents")
}

func (m *membership) SubscribeGossipEvents() <-chan gossipEvent {
	return m.gossipEvents.Subscribe("serfGossipEvents")
}

func (m *membership) Shutdown() error {
	err := m.serf.Leave()
	if err != nil {
		return fmt.Errorf("Failed to leave serf: %w", err)
	}
	return m.serf.Shutdown()
}
func (m *membership) Join(joinAddrs ...string) (int, error) {
	m.Lock()
	defer m.Unlock()
	if m.started {
		return 0, fault.New("Membership already started")
	}
	m.started = true
	m.logger.Info().Msg("Initilizing serf")

	addr, err := net.ResolveTCPAddr("tcp", m.serfAddr)
	if err != nil {
		return 0, fault.Wrap(err, fmsg.With("Failed to resolve serf address"))
	}
	config := serf.DefaultConfig()
	config.MemberlistConfig.BindAddr = addr.IP.String()
	config.MemberlistConfig.BindPort = addr.Port

	m.events = make(chan serf.Event)
	config.EventCh = m.events
	config.Tags, err = m.self.Marshal()
	if err != nil {
		return 0, fault.Wrap(err, fmsg.With("Failed to convert tags to map"))
	}
	config.NodeName = m.self.NodeId

	m.serf, err = serf.Create(config)
	if err != nil {
		return 0, fault.Wrap(err, fmsg.With("Failed to create serf"))
	}

	m.logger.Info().Msg("Config is initialized")

	go m.eventHandler()
	if len(joinAddrs) > 0 {
		m.logger.Info().Strs("addrs", joinAddrs).Msg("Joining serf cluster")
		err = util.Retry(
			func() error {
				successfullyContacted, joinErr := m.serf.Join(joinAddrs, true)
				if joinErr != nil {
					m.logger.Warn().Err(joinErr).Int("successfullyContacted", successfullyContacted).Strs("addrs", joinAddrs).Msg("Failed to join")
				}
				return joinErr
			},
			10,
			func(n int) time.Duration { return time.Duration(n) * time.Second },
		)
		if err != nil {
			return 0, fault.Wrap(err, fmsg.With("Failed to join"))
		}

	}

	return m.serf.Memberlist().NumMembers(), nil
}

func (m *membership) Broadcast(eventType string, payload []byte) error {
	return m.serf.UserEvent(eventType, payload, true)
}
func (m *membership) eventHandler() {

	for e := range m.events {
		ctx := context.Background()

		m.logger.Info().Str("type", e.EventType().String()).Msg("Event")
		switch e.EventType() {
		case serf.EventMemberJoin:
			for _, serfMember := range e.(serf.MemberEvent).Members {

				member, err := memberFromTags(serfMember.Tags)
				if err != nil {
					m.logger.Error().Err(err).Msg("Failed to unmarshal tags")
					continue
				}
				m.joinEvents.Emit(ctx, member)
			}
		case serf.EventMemberLeave, serf.EventMemberFailed:
			for _, serfMember := range e.(serf.MemberEvent).Members {
				member, err := memberFromTags(serfMember.Tags)
				if err != nil {
					m.logger.Error().Err(err).Msg("Failed to unmarshal tags")
					continue
				}
				m.leaveEvents.Emit(ctx, member)
			}
		case serf.EventUser:
			m.gossipEvents.Emit(ctx, gossipEvent{
				event:   e.(serf.UserEvent).Name,
				payload: e.(serf.UserEvent).Payload,
			})
		}

	}
}

func (m *membership) isLocal(member serf.Member) bool {
	return member.Name == m.self.NodeId
}

func (m *membership) Members() ([]Member, error) {
	members := make([]Member, 0)
	for _, serfMember := range m.serf.Members() {
		if serfMember.Status == serf.StatusAlive {
			member, err := memberFromTags(serfMember.Tags)
			if err != nil {
				return nil, fault.Wrap(err, fmsg.With("Failed to unmarshal tags"))
			}
			members = append(members, member)
		}
	}
	return members, nil
}
func (m *membership) Leave() error {
	return m.serf.Leave()
}
