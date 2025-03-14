package membership

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/serf/serf"
	"github.com/unkeyed/unkey/go/pkg/discovery"
	"github.com/unkeyed/unkey/go/pkg/events"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/retry"
)

type Config struct {
	NodeID        string
	AdvertiseHost string
	GossipPort    int
	RpcPort       int
	HttpPort      int
	Logger        logging.Logger
}

type serfMembership struct {
	mu     sync.Mutex
	config Config

	self Member

	serf *serf.Serf

	logger  logging.Logger
	started bool

	onJoin   events.Topic[Member]
	onUpdate events.Topic[Member]
	onLeave  events.Topic[Member]
}

var _ Membership = (*serfMembership)(nil)

// New creates a new membership instance with Serf
func New(config Config) (*serfMembership, error) {

	host, err := parseHost(config.AdvertiseHost)

	if err != nil {
		return nil, err
	}
	// Create self member with metadata
	self := Member{
		NodeID:     config.NodeID,
		Host:       host,
		GossipPort: config.GossipPort,
		RpcPort:    config.RpcPort,
	}

	// Serf configuration
	serfConfig := serf.DefaultConfig()
	serfConfig.NodeName = config.NodeID
	serfConfig.MemberlistConfig.AdvertiseAddr = host
	serfConfig.MemberlistConfig.AdvertisePort = config.GossipPort
	serfConfig.MemberlistConfig.BindAddr = "0.0.0.0"
	serfConfig.MemberlistConfig.BindPort = config.GossipPort

	serfConfig.Tags = self.ToMap()
	serfConfig.LogOutput = logger{config.Logger}

	// Create event handlers for Serf
	eventCh := make(chan serf.Event, 256)
	serfConfig.EventCh = eventCh

	// Create Serf instance
	s, err := serf.Create(serfConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create serf: %w", err)
	}

	m := &serfMembership{
		mu:       sync.Mutex{},
		config:   config,
		logger:   config.Logger,
		started:  false,
		serf:     s,
		self:     self,
		onJoin:   events.NewTopic[Member](100),
		onUpdate: events.NewTopic[Member](100),
		onLeave:  events.NewTopic[Member](100),
	}

	// Process Serf events
	go m.handleEvents(eventCh)

	return m, nil
}

// docker compose gives us weird hostnames that we need to look up first
func parseHost(host string) (string, error) {

	advertiseAddrs, err := net.LookupHost(host)
	if err != nil {
		return "", fmt.Errorf("unable to lookup addr %s: %w", host, err)
	}
	if len(advertiseAddrs) == 0 {
		return "", fmt.Errorf("no advertise addrs found")
	}

	return advertiseAddrs[0], nil
}

// Handle Serf events and propagate to our event system
func (m *serfMembership) handleEvents(ch <-chan serf.Event) {
	for event := range ch {
		switch e := event.(type) {
		case serf.MemberEvent:
			for _, member := range e.Members {
				m.logger.Debug("Received member event",
					"type", e.EventType().String(),
					"nodeID", member.Name,
					"tags", fmt.Sprintf("%v", member.Tags))

				mem, err := memberFromMap(member.Tags)
				if err != nil {
					m.logger.Error(err.Error(),
						"nodeID", member.Name)
					continue
				}

				switch e.EventType() {
				case serf.EventMemberJoin:
					// Don't emit join events for self
					if mem.NodeID != m.self.NodeID {
						m.logger.Debug("Emitting join event", "nodeID", mem.NodeID)
						m.onJoin.Emit(context.Background(), mem)
					}
				case serf.EventMemberLeave, serf.EventMemberFailed:
					m.logger.Debug("Emitting leave event", "nodeID", mem.NodeID)
					m.onLeave.Emit(context.Background(), mem)
				case serf.EventMemberUpdate:
					m.logger.Debug("Emitting update event", "nodeID", mem.NodeID)
					m.onUpdate.Emit(context.Background(), mem)
				case serf.EventMemberReap:
				case serf.EventUser:
				case serf.EventQuery:
				}
			}
		}
	}
}

func (m *serfMembership) SubscribeJoinEvents() <-chan Member {
	return m.onJoin.Subscribe("serfJoinEvents")
}

func (m *serfMembership) SubscribeUpdateEvents() <-chan Member {
	return m.onUpdate.Subscribe("serfUpdateEvents")
}

func (m *serfMembership) SubscribeLeaveEvents() <-chan Member {
	return m.onLeave.Subscribe("serfLeaveEvents")
}

func (m *serfMembership) Leave() error {
	err := m.serf.Leave()
	if err != nil {
		return fmt.Errorf("failed to leave serf: %w", err)
	}
	return m.serf.Shutdown()
}

func (m *serfMembership) Start(discover discovery.Discoverer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return fault.New("Membership already started")
	}

	m.started = true
	m.logger.Info("Initializing serf")

	m.logger.Info("discovering peers")
	addrs, err := discover.Discover()
	if err != nil {
		return fault.Wrap(err)
	}

	// Format addresses to include port - convert ips to ip:port
	joinAddrs := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		// Skip empty addresses
		if addr == "" {
			continue
		}

		joinAddrs = append(joinAddrs, addr)
	}

	if len(joinAddrs) > 0 {
		m.logger.Info("Joining cluster",
			"addrs", strings.Join(joinAddrs, ","),
		)

		err := retry.New(
			retry.Attempts(10),
			retry.Backoff(func(n int) time.Duration { return time.Duration(n) * 100 * time.Millisecond }),
		).Do(
			func() error {
				successfullyContacted, joinErr := m.serf.Join(joinAddrs, true)
				if joinErr != nil {
					m.logger.Warn("failed to join",
						"error", joinErr.Error(),
						"successfullyContacted", successfullyContacted,
						"addrs", strings.Join(joinAddrs, ","),
					)
					return joinErr
				}

				m.logger.Info("Successfully joined cluster",
					"successfullyContacted", successfullyContacted,
					"addrs", strings.Join(joinAddrs, ","),
				)
				return nil
			})

		if err != nil {
			return fault.Wrap(err, fault.WithDesc("Failed to join", ""))
		}
	} else {
		m.logger.Info("No peers to join, starting new cluster")
	}

	return nil
}

func (m *serfMembership) Members() ([]Member, error) {
	members := make([]Member, 0)
	for _, member := range m.serf.Members() {
		if member.Status == serf.StatusAlive {
			m.logger.Debug("Found member",
				"name", member.Name,
				"tags", fmt.Sprintf("%v", member.Tags),
				"status", member.Status.String())

			mem, err := memberFromMap(member.Tags)
			if err != nil {
				return nil, err
			}
			members = append(members, mem)
		}
	}
	return members, nil
}

func (m *serfMembership) Self() Member {
	return m.self
}
