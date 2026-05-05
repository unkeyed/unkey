// Package bus provides a generic, topic-based event bus over a flat gossip
// cluster (one Serf cluster spanning all regions over VPC peering).
//
// The bus exposes three primitives:
//
//   - Publish: named, payload-carrying broadcast with at-least-once delivery
//     and a 60s retention buffer for late joiners.
//   - Subscribe: per-topic handler registration. Handlers must be idempotent;
//     the bus dedupes on (sender_node, event_id) but cannot guarantee
//     exactly-once.
//   - Query: request/response over gossip with an aggregated reply channel
//     and a deadline.
//
// Every payload is wrapped in a BusEnvelope (proto/bus/v1) so that two
// subsystems can evolve their payload schemas independently. The transport
// never inspects the payload bytes.
//
// At-least-once delivery is best-effort: if every replica of a publisher dies
// before the event propagates, the event is lost. The bus is the right choice
// when the source of truth lives elsewhere (the database, a feature flag
// service) and a missed event is recoverable; it is the wrong choice when an
// event must be durably acknowledged across publisher restarts.
package bus
