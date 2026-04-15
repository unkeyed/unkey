/*
Package ratelimit implements lockless distributed rate limiting using a sliding window
algorithm with atomic counters.

# Architecture

All rate limit state is stored in a flat sync.Map of counter entries, keyed by
(name, identifier, duration, sequence). Each entry holds an atomic.Int64 counter
plus a sync.Once that gates the first origin hydration. There are no mutexes in
the hot path:

  - Denials are wait-free: two atomic loads, arithmetic, return.
  - Allows are lock-free: one atomic add after the check.

Local counters are eventually consistent with Redis. Background replay workers push
local increments via INCRBY and CAS-merge the global count back into the local atomic.

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
