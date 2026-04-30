package sinks

import (
	"context"
	"math/rand/v2"
	"time"
)

// sendBackoffs is the schedule between retry attempts on a Sink.Send
// failure. The hardening plan calls for 3 attempts (250ms → 500ms → 1s)
// with full jitter; len(sendBackoffs) is therefore one less than the
// total attempt count, since the schedule only describes the *gaps*
// between attempts. The first delay is short enough that a transient
// network blip does not measurably delay the cursor advance, and the
// final 1s delay keeps the per-batch tail under the 10s poll interval
// even if every retry burns the maximum.
var sendBackoffs = []time.Duration{
	250 * time.Millisecond,
	500 * time.Millisecond,
	1 * time.Second,
}

// maxRetryAfter caps how long SendWithRetry will sleep when honouring a
// provider's Retry-After header. Per-drain cursors made each drain's
// fan-out goroutine independent, so a long sleep on one drain no longer
// pins sibling drains' cursor advance — only the offending drain's own
// next batch is delayed. That removed the strict 10s poll-interval
// ceiling the old 5s cap was sized against.
//
// 60s lines up with the typical Retry-After window providers actually
// emit (Axiom 429s in the 30–60s range) so most rate-limit hints are
// honoured verbatim instead of being clamped to a value the provider
// didn't ask for. The cap is still finite to defend against a
// malicious or buggy provider returning Retry-After: 86400 — at that
// point we'd rather burn the 3-attempt budget and let auto-pause kick
// in than park a worker goroutine for a day.
const maxRetryAfter = 60 * time.Second

// SendWithRetry runs sink.Send up to len(sendBackoffs)+1 times, sleeping
// between attempts. Retry policy: transport failures and HTTP 5xx/408/429
// are retried; everything else (auth, bad request, unknown 4xx) returns
// immediately so a misconfigured drain fails loudly instead of looping
// until auto-pause.
//
// Retry-After is honoured when the provider supplies it, capped at
// maxRetryAfter. Full jitter on the schedule prevents synchronised
// retries from a fleet of replicas all hitting the same provider after
// a shared transient.
//
// Lives in the sinks package because the retry decision is a pure
// function of (Sink, SendError); the coordinator only cares whether a
// batch was delivered.
func SendWithRetry(ctx context.Context, sink Sink, batch []Record) error {
	var lastErr error
	for attempt := range len(sendBackoffs) + 1 {
		err := sink.Send(ctx, batch)
		if err == nil {
			return nil
		}
		lastErr = err
		se := AsSendError(err)
		if !se.Retryable() {
			return err
		}
		if attempt == len(sendBackoffs) {
			break
		}
		// Honour Retry-After when present, but cap it so a malicious
		// or buggy provider can't park the worker indefinitely.
		// Per-drain cursors mean a long sleep on this drain only
		// delays its own next batch; sibling drains in the same group
		// keep advancing.
		var delay time.Duration
		if se.RetryAfter > 0 {
			delay = se.RetryAfter
			if delay > maxRetryAfter {
				delay = maxRetryAfter
			}
		} else {
			delay = jitter(sendBackoffs[attempt])
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return lastErr
}

// jitter returns a duration in [0, d). Full jitter (vs. equal jitter)
// because the AWS architecture-blog measurements show it has the lowest
// total latency when many clients retry against a shared bottleneck —
// which is exactly the shape of N replicas hitting the same provider
// during a regional incident.
func jitter(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(int64(d)))
}
