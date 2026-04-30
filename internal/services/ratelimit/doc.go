/*
Package ratelimit implements lockless distributed rate limiting using a sliding window
algorithm with atomic counters.

# Architecture

All rate limit state is stored in a flat sync.Map of counter entries, keyed by
(workspace, namespace, identifier, duration, sequence). Each entry holds an
atomic.Int64 counter plus a sync.Once that gates the first origin hydration.
There are no mutexes in the hot path:

  - Denials are wait-free: two atomic loads, arithmetic, return.
  - Allows are lock-free: one atomic add after the check.

Local counters are eventually consistent with Redis. Background replay workers push
local increments via INCRBY and CAS-merge the global count back into the local atomic.

# Cross-region propagation

When the service is configured with a DB (see [Config.DB]), denials propagate
across regions through MySQL. On the strict-mode transition (the first denial
in a window), the service writes one row to ratelimit_blocklist; a periodic
sync goroutine on every node reads the active set and inflates the local
counter for each row's originating sequence. Receivers therefore deny the same
identifier without seeing its abusive traffic firsthand. Sliding-window decay
handles the bleed into the next window automatically.

# Known limitation: spread-evenly traffic

The propagation channel is denial-driven: a region only writes to MySQL when
its own local counter trips the limit. Because Redis is per-region in this
deployment, replay only converges counts within a region — there is no
mechanism today that aggregates counts across regions when no single region
hits its limit alone.

Concretely: with N regions and traffic spread perfectly evenly, each region
sees total/N. If the customer's limit is L and total is X·L globally, each
region locally sees X·L/N. For X < N, no region's local count ever crosses
L, so no region denies, and the propagation channel never fires. The customer
effectively gets up to N·L globally before any single region trips. This
matters most for low-N attacks against high-replica deployments.

Closing this needs a different mechanism than denial propagation — periodic
per-region count reporting through the same MySQL channel, with receivers
summing across regions and adding the remote contribution into the
sliding-window math. That work is deferred; the current channel handles the
"burst to one region" case (the common attack shape) correctly.

# Rate Limit Algorithm

The sliding window implementation:

 1. Computes sequence numbers for current and previous windows from the request time.
 2. Loads both atomic counters from the sync.Map (creating them if needed).
 3. Calculates effective request count: 100% of current window + weighted portion
    of previous window based on elapsed time in the current window.
 4. If the effective count exceeds the limit, denies the request.
 5. Otherwise, atomically increments the current window counter and buffers
    the request for async replay to Redis.

# Thread Safety

All operations are safe for concurrent use without external synchronization.
The service uses sync.Map for counter storage and sync/atomic for counter updates.

# Error Handling

The service handles various error conditions:
  - Invalid configurations (empty identifiers, zero limits, short durations)
  - Redis unavailability (continues with local-only decisions)
  - Circuit breaker trips during sustained Redis failures

See the [RatelimitRequest] and [RatelimitResponse] types for the API contract.
*/
package ratelimit
