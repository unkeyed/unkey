package bus

import (
	"time"

	"github.com/hashicorp/serf/serf"
	busv1 "github.com/unkeyed/unkey/gen/proto/bus/v1"
	"github.com/unkeyed/unkey/pkg/bus/metrics"
	"github.com/unkeyed/unkey/pkg/logger"
	"google.golang.org/protobuf/proto"
)

// replayQueryName is the Serf query name used for the on-join replay
// protocol. It is reserved: the public Bus.Query method does not enforce
// this, but callers must avoid using this exact name.
const replayQueryName = "bus.replay"

// replayQueryTimeout caps the total time a replay-on-join query waits for
// a single peer's response. The targeted peer needs to scan its replay log
// and wire the response across regions; 5 s is generous for that and
// short enough that a slow or absent peer does not delay further joins.
const replayQueryTimeout = 5 * time.Second

// onMembersJoined fires a replay-on-join query to each newly observed peer
// that is not the local node. Coalesced join events arrive with multiple
// members at once during cluster bootstrap; each gets its own query.
//
// The query is targeted at one peer at a time (FilterNodes) so the
// response set is bounded and replay traffic does not amplify by the
// fanout factor.
func (b *serfBus) onMembersJoined(members []serf.Member) {
	for _, m := range members {
		if m.Name == b.cfg.NodeID {
			continue
		}
		b.wg.Add(1)
		go b.requestReplayFrom(m.Name)
	}
}

func (b *serfBus) requestReplayFrom(peer string) {
	defer b.wg.Done()

	b.mu.RLock()
	since := b.maxSeenAtMs[peer]
	b.mu.RUnlock()

	req := &busv1.BusReplayRequest{SinceMs: since}
	payload, err := proto.Marshal(req)
	if err != nil {
		logger.Warn("Bus: failed to marshal replay request", "peer", peer, "error", err)
		return
	}

	params := b.serf.DefaultQueryParams()
	params.FilterNodes = []string{peer}
	params.Timeout = replayQueryTimeout
	// RequestAck would add a return trip per peer for ack-channel polling
	// that we don't read. Disable to keep the query lean.
	params.RequestAck = false

	resp, err := b.serf.Query(replayQueryName, payload, params)
	if err != nil {
		metrics.ReplayRequestsTotal.WithLabelValues("query_error").Inc()
		logger.Warn("Bus: replay query failed", "peer", peer, "error", err)
		return
	}

	respCh := resp.ResponseCh()
	for {
		select {
		case <-b.done:
			return
		case nr, ok := <-respCh:
			if !ok {
				return
			}
			if nr.From != peer {
				// FilterNodes should keep this constrained to the targeted
				// peer, but be defensive: a stray response would re-feed
				// envelopes the cursor accounting cannot reason about.
				continue
			}
			b.applyReplayResponse(peer, nr.Payload)
		}
	}
}

func (b *serfBus) applyReplayResponse(peer string, payload []byte) {
	resp := &busv1.BusReplayResponse{}
	if err := proto.Unmarshal(payload, resp); err != nil {
		metrics.ReplayRequestsTotal.WithLabelValues("decode_error").Inc()
		logger.Warn("Bus: failed to unmarshal replay response", "peer", peer, "error", err)
		return
	}

	if len(resp.Envelopes) == 0 {
		metrics.ReplayRequestsTotal.WithLabelValues("empty").Inc()
		return
	}

	metrics.ReplayRequestsTotal.WithLabelValues("applied").Inc()

	for _, envBytes := range resp.Envelopes {
		// The wire envelope carries its own topic; re-feed through the
		// normal dispatch path so dedup, pause gating, and metrics behave
		// identically to a live user event.
		envelope := &busv1.BusEnvelope{}
		if err := proto.Unmarshal(envBytes, envelope); err != nil {
			logger.Warn("Bus: skipping malformed envelope in replay", "peer", peer, "error", err)
			continue
		}
		b.dispatchEnvelope(envelope.Topic, envBytes)
	}
}

// handleReplayQuery is the responder side. The targeted peer scans its own
// replay log for envelopes published more recently than the cursor and
// returns them inline.
func (b *serfBus) handleReplayQuery(q *serf.Query) {
	req := &busv1.BusReplayRequest{}
	if err := proto.Unmarshal(q.Payload, req); err != nil {
		logger.Warn("Bus: failed to unmarshal replay request", "error", err)
		return
	}

	envelopes := b.collectReplayEnvelopes(req.SinceMs)

	resp := &busv1.BusReplayResponse{Envelopes: envelopes}
	out, err := proto.Marshal(resp)
	if err != nil {
		logger.Warn("Bus: failed to marshal replay response", "error", err)
		return
	}

	if err := q.Respond(out); err != nil {
		logger.Warn("Bus: failed to respond to replay query", "error", err, "source", q.SourceNode())
	}
}

func (b *serfBus) collectReplayEnvelopes(sinceMs int64) [][]byte {
	// Iterate every topic ring; an envelope in the log was published by
	// this node, so we don't need to re-check sender_node here.
	b.replay.mu.Lock()
	defer b.replay.mu.Unlock()

	out := make([][]byte, 0)
	for _, ring := range b.replay.rings {
		for e := ring.entries.Front(); e != nil; e = e.Next() {
			entry := e.Value.(replayEntry)
			if sinceMs > 0 && entry.at.UnixMilli() <= sinceMs {
				continue
			}
			out = append(out, entry.envelope)
		}
	}
	return out
}
