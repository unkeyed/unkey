package bus

import (
	"context"
	"errors"

	"google.golang.org/protobuf/proto"
)

// ErrBusPaused is returned by Publish when the bus has been paused via the
// admin endpoint. Subscriber dispatch is also gated; see (Bus).Pause.
var ErrBusPaused = errors.New("bus is paused")

// ErrPayloadTooLarge is returned by Publish when the marshalled envelope
// exceeds the configured MaxUserEventSize for the underlying transport.
var ErrPayloadTooLarge = errors.New("bus payload exceeds MaxUserEventSize")

// Event is the deserialized form of a published message as seen by a
// subscriber. It mirrors the BusEnvelope proto with field names that read
// naturally in Go.
//
// SenderNode and ID together form the dedup key. Handlers receive the same
// (SenderNode, ID) at most once per receiver, even if Serf retransmits or a
// replay query refills the same event from a publisher's ring.
type Event struct {
	// Topic the event was published on.
	Topic string

	// Unique event id chosen by the publisher.
	ID string

	// Region of the publisher.
	SourceRegion string

	// Node id of the publisher.
	SenderNode string

	// Publish-time timestamp in unix milliseconds.
	SentAtMs int64

	// Opaque payload bytes. The topic owner decodes against its own schema.
	Payload []byte
}

// Member is a snapshot of a single peer in the cluster. It does not leak the
// underlying memberlist/Serf node type so that callers in svc/* can depend on
// pkg/bus alone.
type Member struct {
	// NodeID is the unique id this member registered with at startup.
	NodeID string

	// Addr is the advertise address (e.g. "10.0.4.7:7946"). For pods inside a
	// peered VPC, this is the pod IP and is directly dialable.
	Addr string

	// Tags are the key=value labels the member published at join time:
	// role, region, version, instance.
	Tags map[string]string
}

// Handler is invoked once per (SenderNode, ID) per subscription. It must not
// block the caller for long; offload heavy work to a goroutine if needed.
//
// Handlers must be idempotent. The dedup cache is bounded; under sustained
// duplication pressure or a long partition recovery, the same event may
// appear twice.
type Handler func(Event)

// QueryResponse is one reply to a Query, as it arrives over the channel.
type QueryResponse struct {
	// From is the responding peer's NodeID.
	From string

	// Payload is the responder-supplied bytes. Decoded by the caller.
	Payload []byte
}

// Bus is the public event bus surface. The same interface is satisfied by
// the noop implementation (bus.NewNoop) and the Serf-backed implementation
// (bus.New).
type Bus interface {
	// Publish marshals payload into a BusEnvelope under topic and broadcasts
	// it. Returns ErrBusPaused if the bus is paused, or ErrPayloadTooLarge if
	// the wire size exceeds MaxUserEventSize.
	Publish(ctx context.Context, topic string, payload proto.Message) error

	// Subscribe registers a handler for topic. The returned function
	// unregisters the handler. Multiple handlers may register against the
	// same topic; each is invoked once per deduped event.
	Subscribe(topic string, handler Handler) (unsubscribe func())

	// Query fans out a request to all members (optionally tag-filtered by the
	// underlying implementation) and returns a channel that receives replies
	// as they arrive. The channel is closed when the context deadline fires
	// or when Serf signals all expected responses have arrived, whichever
	// comes first. Callers must not assume a specific number of responses.
	Query(ctx context.Context, topic string, payload proto.Message) (<-chan QueryResponse, error)

	// Members returns a snapshot of currently known cluster members.
	Members() []Member

	// Pause stops Publish from broadcasting and gates subscriber dispatch
	// (incoming events are dropped with reason="paused"). Membership
	// keep-alives continue so the pod stays in the cluster; Resume restores
	// normal operation without a rejoin.
	//
	// The kill switch for incident response.
	Pause()

	// Resume re-enables Publish and dispatch after a Pause.
	Resume()

	// IsPaused reports whether the bus is currently paused. The same state
	// is also surfaced as the bus_paused gauge for alerting; this method
	// exists for the admin status endpoint and tests.
	IsPaused() bool

	// Close gracefully leaves the cluster and releases resources.
	Close() error
}
