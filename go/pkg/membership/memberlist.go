package membership

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/unkeyed/unkey/go/pkg/discovery"
	"github.com/unkeyed/unkey/go/pkg/events"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/logging"
	"github.com/unkeyed/unkey/go/pkg/retry"
)

// Config specifies the configuration options for creating a new membership instance.
type Config struct {
	// NodeID is the unique identifier for this node
	NodeID string
	// AdvertiseAddr is the ip or dns name where this node can be
	AdvertiseAddr string
	// GossipPort is the port used for cluster membership gossip protocol
	GossipPort int
	// Logger is the logging interface used for membership-related logs
	Logger logging.Logger
}

type membership struct {
	mu     sync.Mutex
	config Config

	self Member

	bus        *bus
	memberlist *memberlist.Memberlist

	logger  logging.Logger
	started bool
}

var _ Membership = (*membership)(nil)

// New creates a new membership instance with the provided configuration.
// It initializes the memberlist with default LAN configuration and sets up event buses.
// Returns the new membership instance and any error encountered during creation.
func New(config Config) (*membership, error) {

	b := &bus{
		onJoin:   events.NewTopic[Member](),
		onLeave:  events.NewTopic[Member](),
		onUpdate: events.NewTopic[Member](),
	}

	advertiseAddrs, err := net.LookupHost(config.AdvertiseAddr)
	if err != nil {
		return nil, fmt.Errorf("unable to lookup addr %s: %w", config.AdvertiseAddr, err)
	}
	if len(advertiseAddrs) == 0 {
		return nil, fmt.Errorf("no advertise addrs found")
	}
	memberlistConfig := memberlist.DefaultLANConfig()
	memberlistConfig.Name = config.NodeID
	memberlistConfig.AdvertiseAddr = advertiseAddrs[0]
	memberlistConfig.AdvertisePort = config.GossipPort
	memberlistConfig.BindPort = config.GossipPort
	memberlistConfig.Events = b
	memberlistConfig.LogOutput = logger{config.Logger}

	list, err := memberlist.Create(memberlistConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create memberlist: %w", err)
	}

	m := &membership{
		mu:         sync.Mutex{},
		config:     config,
		logger:     config.Logger,
		started:    false,
		memberlist: list,
		self: Member{
			NodeID: config.NodeID,
			Addr:   fmt.Sprintf("%s:%d", config.AdvertiseAddr, config.GossipPort),
		},
		bus: b,
	}

	return m, nil
}

func (m *membership) SubscribeJoinEvents() <-chan Member {
	return m.bus.onJoin.Subscribe("serfJoinEvents")
}

func (m *membership) SubscribeLeaveEvents() <-chan Member {
	return m.bus.onLeave.Subscribe("serfLeaveEvents")
}

func (m *membership) Leave() error {
	err := m.memberlist.Leave(time.Second * 15)
	if err != nil {
		return fmt.Errorf("Failed to leave serf: %w", err)
	}
	return m.memberlist.Shutdown()
}

func (m *membership) Start(discover discovery.Discoverer) error {

	ctx := context.Background()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started {
		return fault.New("Membership already started")
	}
	m.started = true
	m.logger.Info(ctx, "Initilizing memberlist")

	m.logger.Info(ctx, "discovering peers")
	addrs, err := discover.Discover()
	if err != nil {
		return fault.Wrap(err)
	}

	if len(addrs) > 0 {
		m.logger.Info(ctx, "Joining cluster", slog.String("addrs", strings.Join(addrs, ",")))
		err := retry.New(
			retry.Attempts(10),
			retry.Backoff(func(n int) time.Duration { return time.Duration(n) * time.Second }),
		).Do(
			func() error {
				successfullyContacted, joinErr := m.memberlist.Join(addrs)
				if joinErr != nil {
					m.logger.Warn(ctx,
						"failed to join",
						slog.String("error", joinErr.Error()),
						slog.Int("successfullyContacted", successfullyContacted),
						slog.String("addrs", strings.Join(addrs, ",")),
					)
				}
				return joinErr
			})
		if err != nil {
			return fault.Wrap(err, fault.WithDesc("Failed to join", ""))
		}

	}

	return nil
}

func (m *membership) Members() ([]Member, error) {
	members := make([]Member, 0)
	for _, m := range m.memberlist.Members() {
		if m.State == memberlist.StateAlive {
			members = append(members, Member{
				NodeID: m.Name,
				Addr:   m.Addr.String(),
			})
		}
	}
	return members, nil
}

func (m *membership) Self() Member {

	return m.self
}
