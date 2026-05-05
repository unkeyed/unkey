package bus

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/serf/serf"
	busv1 "github.com/unkeyed/unkey/gen/proto/bus/v1"
	"github.com/unkeyed/unkey/pkg/bus/metrics"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	"google.golang.org/protobuf/proto"
)

// eventChannelBuffer sizes the buffered channel Serf delivers events on.
// Sized for a small burst (a member-update fanout in a 100-pod cluster
// produces a few dozen events at once) without dropping. If sustained
// pressure causes Serf to fill this buffer, it logs and the events are
// nevertheless delivered through anti-entropy on the next gossip round.
const eventChannelBuffer = 256

// reconnectInterval is how often the join loop re-checks the cluster for
// isolation and retries seed dial. Matches pkg/cluster's interval for
// consistency.
const reconnectInterval = 30 * time.Second

// initialJoinBackoff is the starting delay between failed join attempts at
// startup. It doubles on each failure up to reconnectInterval.
const initialJoinBackoff = 500 * time.Millisecond

// serfBus is the production Bus implementation backed by hashicorp/serf.
//
// Lifecycle: New starts a Serf agent, an event loop, a metrics goroutine,
// and a join-maintenance goroutine. Close cancels all four and Leaves the
// cluster.
type serfBus struct {
	cfg     Config
	serf    *serf.Serf
	eventCh chan serf.Event

	mu   sync.RWMutex
	subs map[string][]*subscription

	// maxSeenAtMs tracks the highest BusEnvelope.SentAtMs observed per
	// sender node. Used as the cursor for replay-on-join queries: when
	// a peer (re)joins, we ask it for events newer than this. Protected
	// by mu.
	maxSeenAtMs map[string]int64

	paused    atomic.Bool
	closing   atomic.Bool
	rejoining atomic.Bool
	done      chan struct{}
	wg        sync.WaitGroup

	dedup  *dedupCache
	replay *replayLog
}

type subscription struct {
	handler Handler
}

var _ Bus = (*serfBus)(nil)

// New creates a Serf agent bound to cfg.BindAddr:cfg.BindPort, joins the
// supplied seeds in the background, and returns a ready-to-use Bus. Returns
// an error if the agent cannot start at all (port in use, key length
// invalid, etc.); seed-join failures are retried internally and never cause
// New to return an error.
func New(cfg Config) (Bus, error) {
	cfg.setDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	eventCh := make(chan serf.Event, eventChannelBuffer)

	serfCfg := serf.DefaultConfig()
	serfCfg.NodeName = cfg.NodeID
	serfCfg.Tags = cfg.Tags
	serfCfg.EventCh = eventCh
	serfCfg.LogOutput = newLogWriter("serf")
	serfCfg.UserEventSizeLimit = cfg.MaxUserEventSize
	// Use WAN timings: this cluster spans regions and trans-Atlantic RTT
	// would trigger false-positive failure detection on LAN defaults.
	serfCfg.MemberlistConfig = memberlist.DefaultWANConfig()
	serfCfg.MemberlistConfig.Name = cfg.NodeID
	serfCfg.MemberlistConfig.BindAddr = cfg.BindAddr
	serfCfg.MemberlistConfig.BindPort = cfg.BindPort
	serfCfg.MemberlistConfig.AdvertisePort = cfg.BindPort
	if cfg.AdvertiseAddr != "" {
		serfCfg.MemberlistConfig.AdvertiseAddr = cfg.AdvertiseAddr
	}
	if len(cfg.SecretKey) > 0 {
		serfCfg.MemberlistConfig.SecretKey = cfg.SecretKey
	}
	serfCfg.MemberlistConfig.LogOutput = newLogWriter("memberlist")

	s, err := serf.Create(serfCfg)
	if err != nil {
		return nil, fmt.Errorf("bus: serf.Create: %w", err)
	}

	b := &serfBus{
		cfg:         cfg,
		serf:        s,
		eventCh:     eventCh,
		mu:          sync.RWMutex{},
		subs:        make(map[string][]*subscription),
		maxSeenAtMs: make(map[string]int64),
		paused:      atomic.Bool{},
		closing:     atomic.Bool{},
		rejoining:   atomic.Bool{},
		done:        make(chan struct{}),
		wg:          sync.WaitGroup{},
		dedup:       newDedupCache(cfg.DedupCacheSize),
		replay:      nil, // set below
	}
	b.replay = newReplayLog(cfg.ReplayLogBytesPerTopic, cfg.ReplayLogBytesTotal, replayHooks{
		onTopicEviction: func(topic string) {
			metrics.ReplayLogEvictionsTotal.WithLabelValues(topic, "topic_cap").Inc()
		},
		onAggregateEviction: func(topic string) {
			metrics.ReplayLogEvictionsTotal.WithLabelValues(topic, "aggregate_cap").Inc()
		},
		onTopicSize: func(topic string, used int) {
			metrics.ReplayLogBytesUsed.WithLabelValues(topic).Set(float64(used))
		},
	})

	// Seed the gauge with the local node so the metric is non-zero before
	// the first peer arrives. handleEvent updates it on every member event
	// after that — no polling goroutine.
	metrics.MembersCount.WithLabelValues(b.cfg.Region).Set(float64(b.serf.NumNodes()))

	b.wg.Add(1)
	go b.eventLoop()

	if len(cfg.Seeds) > 0 {
		b.wg.Add(1)
		go b.initialJoin()
	}

	return b, nil
}

// Publish marshals payload into a BusEnvelope and broadcasts it as a Serf
// user event named after the topic. The envelope is also appended to the
// per-topic replay log so a peer that joins within the retention window can
// recover the event via PR 3's replay query.
func (b *serfBus) Publish(_ context.Context, topic string, payload proto.Message) error {
	if b.paused.Load() {
		metrics.EventsDroppedTotal.WithLabelValues("paused").Inc()
		return ErrBusPaused
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return fmt.Errorf("bus: marshal payload: %w", err)
	}

	envelope := &busv1.BusEnvelope{
		Topic:        topic,
		Id:           uid.New("", 16),
		SourceRegion: b.cfg.Region,
		SenderNode:   b.cfg.NodeID,
		SentAtMs:     time.Now().UnixMilli(),
		Payload:      payloadBytes,
	}

	envelopeBytes, err := proto.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("bus: marshal envelope: %w", err)
	}

	if len(envelopeBytes)+len(topic) > b.cfg.MaxUserEventSize {
		metrics.EventsDroppedTotal.WithLabelValues("payload_too_large").Inc()
		return ErrPayloadTooLarge
	}

	b.replay.Append(replayEntry{
		topic:    topic,
		sender:   b.cfg.NodeID,
		id:       envelope.Id,
		envelope: envelopeBytes,
		at:       time.Now(),
	})

	if err := b.serf.UserEvent(topic, envelopeBytes, false); err != nil {
		metrics.EventsDroppedTotal.WithLabelValues("publish_error").Inc()
		return fmt.Errorf("bus: serf user event: %w", err)
	}

	metrics.EventsPublishedTotal.WithLabelValues(topic).Inc()
	return nil
}

// Subscribe registers a handler for the given topic. The returned function
// removes this subscription; it is safe to call multiple times.
func (b *serfBus) Subscribe(topic string, handler Handler) func() {
	if handler == nil {
		return func() {}
	}

	sub := &subscription{handler: handler}

	b.mu.Lock()
	b.subs[topic] = append(b.subs[topic], sub)
	b.mu.Unlock()

	var unsubOnce sync.Once
	return func() {
		unsubOnce.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			subs := b.subs[topic]
			for i, s := range subs {
				if s == sub {
					b.subs[topic] = append(subs[:i], subs[i+1:]...)
					break
				}
			}
			if len(b.subs[topic]) == 0 {
				delete(b.subs, topic)
			}
		})
	}
}

// Query fans out a Serf query named after topic with the marshalled payload
// and returns a channel of replies. The channel is closed when Serf signals
// the query is finished or when the caller's context is cancelled.
func (b *serfBus) Query(ctx context.Context, topic string, payload proto.Message) (<-chan QueryResponse, error) {
	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("bus: marshal query payload: %w", err)
	}

	params := b.serf.DefaultQueryParams()
	if dl, ok := ctx.Deadline(); ok {
		if remaining := time.Until(dl); remaining > 0 {
			params.Timeout = remaining
		}
	}

	resp, err := b.serf.Query(topic, payloadBytes, params)
	if err != nil {
		return nil, fmt.Errorf("bus: serf query: %w", err)
	}

	out := make(chan QueryResponse)
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		defer close(out)
		respCh := resp.ResponseCh()
		for {
			select {
			case <-ctx.Done():
				return
			case <-b.done:
				return
			case nr, ok := <-respCh:
				if !ok {
					return
				}
				select {
				case out <- QueryResponse{From: nr.From, Payload: nr.Payload}:
				case <-ctx.Done():
					return
				case <-b.done:
					return
				}
			}
		}
	}()

	return out, nil
}

// Members returns a snapshot of currently-known cluster members in
// transport-agnostic form.
func (b *serfBus) Members() []Member {
	raw := b.serf.Members()
	out := make([]Member, 0, len(raw))
	for _, m := range raw {
		if m.Status != serf.StatusAlive {
			continue
		}
		addr := fmt.Sprintf("%s:%d", m.Addr.String(), m.Port)
		tags := make(map[string]string, len(m.Tags))
		for k, v := range m.Tags {
			tags[k] = v
		}
		out = append(out, Member{NodeID: m.Name, Addr: addr, Tags: tags})
	}
	return out
}

// Pause stops Publish and gates dispatch. Membership keep-alives continue
// so the pod stays in the cluster; Resume restores normal operation
// without a rejoin.
func (b *serfBus) Pause() {
	if !b.paused.Swap(true) {
		metrics.Paused.Set(1)
		logger.Warn("Bus paused")
	}
}

// Resume re-enables Publish and dispatch.
func (b *serfBus) Resume() {
	if b.paused.Swap(false) {
		metrics.Paused.Set(0)
		logger.Info("Bus resumed")
	}
}

// IsPaused reports the current pause state.
func (b *serfBus) IsPaused() bool {
	return b.paused.Load()
}

// Close gracefully leaves the cluster and shuts down. Safe to call multiple
// times; only the first call performs the shutdown.
func (b *serfBus) Close() error {
	if alreadyClosing := b.closing.Swap(true); alreadyClosing {
		return nil
	}
	close(b.done)

	if err := b.serf.Leave(); err != nil {
		logger.Warn("Bus: Serf Leave returned error", "error", err)
	}
	if err := b.serf.Shutdown(); err != nil {
		return fmt.Errorf("bus: serf shutdown: %w", err)
	}

	b.wg.Wait()
	return nil
}

// eventLoop is the single consumer of Serf's event channel. It dispatches
// user events to subscribers, observes member events for metrics, and
// drops queries (PR 3 will register a handler for replay queries).
func (b *serfBus) eventLoop() {
	defer b.wg.Done()

	for {
		select {
		case <-b.done:
			return
		case ev, ok := <-b.eventCh:
			if !ok {
				return
			}
			b.handleEvent(ev)
		}
	}
}

func (b *serfBus) handleEvent(ev serf.Event) {
	switch e := ev.(type) {
	case serf.UserEvent:
		b.dispatchEnvelope(e.Name, e.Payload)
	case serf.MemberEvent:
		metrics.MembershipEventsTotal.WithLabelValues(e.Type.String()).Add(float64(len(e.Members)))
		// The current member count is authoritative; reset the gauge from
		// it on every coalesced batch so the metric stays accurate without
		// a separate polling loop.
		metrics.MembersCount.WithLabelValues(b.cfg.Region).Set(float64(b.serf.NumNodes()))
		if e.Type == serf.EventMemberJoin {
			b.onMembersJoined(e.Members)
		}
		// On leave/failed events that drop us back to a single-node cluster,
		// kick a one-shot rejoin against the seeds. This replaces the
		// 30-second polling check that used to live in joinLoop.
		if (e.Type == serf.EventMemberLeave || e.Type == serf.EventMemberFailed) &&
			len(b.cfg.Seeds) > 0 && b.serf.NumNodes() <= 1 {
			b.triggerRejoin()
		}
	case *serf.Query:
		if e.Name == replayQueryName {
			b.handleReplayQuery(e)
		}
		// Other inbound queries are ignored until a public handler-
		// registration API exists; the initiator's deadline will fire.
	}
}

// dispatchEnvelope is the single inbound dispatch path. It is called both
// from the Serf event loop for live user events and from the replay-on-join
// re-feed in replay.go, so the dedup, pause gate, and metrics behave
// identically for both sources.
func (b *serfBus) dispatchEnvelope(topic string, envelopeBytes []byte) {
	envelope := &busv1.BusEnvelope{}
	if err := proto.Unmarshal(envelopeBytes, envelope); err != nil {
		metrics.EventsReceivedTotal.WithLabelValues(topic, "decode_error").Inc()
		logger.Warn("Bus: failed to unmarshal envelope", "topic", topic, "error", err)
		return
	}

	if envelope.SenderNode == b.cfg.NodeID {
		// Self-events are filtered before dispatch. Without this, every
		// publisher would observe its own events and have to defensively
		// check SenderNode in every handler.
		return
	}

	if b.dedup.SeenOrAdd(dedupKey{sender: envelope.SenderNode, id: envelope.Id}) {
		metrics.EventsReceivedTotal.WithLabelValues(topic, "deduped").Inc()
		return
	}

	b.recordSeen(envelope.SenderNode, envelope.SentAtMs)

	if b.paused.Load() {
		metrics.EventsDroppedTotal.WithLabelValues("paused").Inc()
		return
	}

	if envelope.SentAtMs > 0 {
		latency := time.Since(time.UnixMilli(envelope.SentAtMs)).Seconds()
		metrics.EventLatencySeconds.
			WithLabelValues(topic, envelope.SourceRegion, b.cfg.Region).
			Observe(latency)
	}

	b.mu.RLock()
	handlers := append([]*subscription(nil), b.subs[topic]...)
	b.mu.RUnlock()

	if len(handlers) == 0 {
		metrics.EventsReceivedTotal.WithLabelValues(topic, "no_handler").Inc()
		return
	}

	evt := Event{
		Topic:        topic,
		ID:           envelope.Id,
		SourceRegion: envelope.SourceRegion,
		SenderNode:   envelope.SenderNode,
		SentAtMs:     envelope.SentAtMs,
		Payload:      envelope.Payload,
	}
	for _, sub := range handlers {
		sub.handler(evt)
	}
	metrics.EventsReceivedTotal.WithLabelValues(topic, "handled").Inc()
}

// recordSeen advances the per-sender high-water mark of observed publish
// timestamps. It feeds the cursor passed in replay-on-join queries so we
// only ask for events newer than what we already have.
func (b *serfBus) recordSeen(sender string, sentAtMs int64) {
	if sender == "" || sentAtMs <= 0 {
		return
	}
	b.mu.Lock()
	if existing := b.maxSeenAtMs[sender]; sentAtMs > existing {
		b.maxSeenAtMs[sender] = sentAtMs
	}
	b.mu.Unlock()
}

// initialJoin runs once at startup, retrying with exponential backoff until
// the seed list is reachable. After the first successful join, the goroutine
// exits; isolation recovery is event-driven (see handleEvent →
// triggerRejoin), so there is no periodic poll.
func (b *serfBus) initialJoin() {
	defer b.wg.Done()

	backoff := initialJoinBackoff
	for {
		select {
		case <-b.done:
			return
		default:
		}

		_, err := b.serf.Join(b.cfg.Seeds, true)
		if err == nil {
			metrics.SeedJoinAttemptsTotal.WithLabelValues("success").Inc()
			logger.Info("Bus joined seeds", "seeds", b.cfg.Seeds)
			return
		}

		metrics.SeedJoinAttemptsTotal.WithLabelValues("failure").Inc()
		logger.Warn("Bus seed join failed, retrying",
			"error", err, "seeds", b.cfg.Seeds, "next_backoff", backoff)

		select {
		case <-b.done:
			return
		case <-time.After(backoff):
		}

		backoff = min(backoff*2, reconnectInterval)
	}
}

// triggerRejoin runs a single-flight rejoin in the background. Called from
// handleEvent when a member-leave/failed drops us to an isolated cluster,
// so we react in the same instant Serf observes the topology change rather
// than waiting for a periodic poll.
func (b *serfBus) triggerRejoin() {
	if !b.rejoining.CompareAndSwap(false, true) {
		return
	}
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		defer b.rejoining.Store(false)

		logger.Warn("Bus is isolated, attempting to rejoin", "seeds", b.cfg.Seeds)
		if _, err := b.serf.Join(b.cfg.Seeds, true); err != nil {
			metrics.SeedJoinAttemptsTotal.WithLabelValues("failure").Inc()
			logger.Warn("Bus rejoin failed", "error", err, "seeds", b.cfg.Seeds)
			return
		}
		metrics.SeedJoinAttemptsTotal.WithLabelValues("success").Inc()
	}()
}

