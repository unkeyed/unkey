package membership

import (
	"context"
	"fmt"
	"log/slog"
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

type Config struct {
	NodeID     string
	Addr       string
	GossipPort int
	Logger     logging.Logger
}

type membership struct {
	mu sync.Mutex

	self Member

	bus        *bus
	memberlist *memberlist.Memberlist

	logger  logging.Logger
	started bool
}

var _ Membership = (*membership)(nil)

func New(config Config) (*membership, error) {

	b := &bus{
		onJoin:   events.NewTopic[Member](),
		onLeave:  events.NewTopic[Member](),
		onUpdate: events.NewTopic[Member](),
	}

	memberlistConfig := memberlist.DefaultLANConfig()
	memberlistConfig.Name = config.NodeID
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
		logger:     config.Logger,
		started:    false,
		memberlist: list,
		self: Member{
			NodeID: config.NodeID,
			Addr:   config.Addr,
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
