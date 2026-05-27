/*
Package ratelimit implements lockless distributed rate limiting using a sliding window
algorithm with atomic counters.

# Architecture

All rate limit state is stored in a flat sync.Map of counter entries, keyed by
(workspace, namespace, identifier, duration, sequence). Each entry holds an
atomic.Int64 counter plus a sync.Once that gates the first origin sync. Warm
entries periodically refresh from origin to prevent idle replicas from serving
stale long-window state indefinitely. There are no mutexes in the hot path:

  - Denials are wait-free: two atomic loads, arithmetic, return.
  - Single checks are lock-free: a bounded CAS loop commits the increment.
  - Batch checks use optimistic atomic adds and roll back the full batch on failure.

Local counters are eventually consistent with Redis. Background replay workers push
local increments via INCRBY and CAS-merge the global count back into the local atomic.

# Cross-region propagation

When the service is configured with a DB (see [Config.DB]), regions share
sliding-window counters through ratelimit_global_counters. Each region periodically
flushes its own observed count for active window cells, keyed by region. Each
region also periodically imports the sum of other regions' rows and folds that
foreign count into the local sliding-window math.

The push path has two write-reduction filters. It only emits entries whose
current local count has changed since the last successful push, and whose
observed count has reached globalUtilizationFloor of the request limit. This
keeps low-value, low-utilization counters from turning into MySQL write load
while still sharing counts once they can affect another region's decision.
The floor is stored as a threshold rather than a request-path latch so regional
Redis convergence can make an entry eligible after the request that created it.

# Rate limit algorithm

The sliding window implementation:

 1. Computes sequence numbers for current and previous windows from the request time.
 2. Loads both atomic counters from the sync.Map (creating them if needed).
 3. Calculates effective request count: 100% of current window + weighted portion
    of previous window based on elapsed time in the current window.
 4. If the effective count exceeds the limit, denies the request.
 5. Otherwise, atomically commits the current-window increment and buffers the
    request for async replay to Redis.

For [Service.RatelimitMany], the package applies all increments first, evaluates
every requested limit against the post-increment values, then either keeps the
entire batch or rolls every increment back. The cross-region push subtracts those
speculative increments while a batch is in flight, so a batch that rolls back
cannot publish temporary state to ratelimit_global_counters.

# Thread safety

All operations are safe for concurrent use without external synchronization.
The service uses sync.Map for counter storage and sync/atomic for counter updates.

# Error handling

The service handles various error conditions:
  - Invalid configurations (empty identifiers, zero limits, short durations)
  - Redis unavailability (continues with local-only decisions)
  - Circuit breaker trips during sustained Redis failures
  - MySQL unavailability for cross-region propagation (continues with local and
    Redis-backed regional decisions)

See the [RatelimitRequest] and [RatelimitResponse] types for the API contract.
*/
package ratelimit
