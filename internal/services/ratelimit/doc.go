/*
Package ratelimit implements a distributed rate limiting system using a sliding window algorithm
with cluster-wide state synchronization. It provides precise rate limiting across multiple nodes
while maintaining low latency through local decision making and asynchronous state propagation.

# Architecture

The rate limiter uses a sliding window algorithm with the following key components:

  - Buckets: Track rate limit state for each unique identifier+limit+duration combination
  - Windows: Time-based counters that slide to maintain accurate request counts
  - Origin Nodes: Designated by consistent hashing to be the source of truth for each identifier
  - State Propagation: Asynchronous updates to maintain eventual consistency across the cluster

# Rate Limit Algorithm

The sliding window implementation:

 1. Maintains separate counters for current and previous time windows
 2. Calculates effective request count by combining:
    - 100% of current window
    - Weighted portion of previous window based on elapsed time
 3. Uses consistent hashing to route requests to origin nodes
 4. Propagates state changes asynchronously to maintain cluster-wide consistency

# Usage

To create a new rate limiting service:

	svc, err := ratelimit.New(ratelimit.Config{
	    Cluster: cluster,
	    Clock:   clock,
	})

To check if a request is allowed:

	resp, err := svc.Ratelimit(ctx, RatelimitRequest{
	    Identifier: "user-123",
	    Limit:      100,
	    Duration:   time.Minute,
	    Cost:      1,
	})

# Thread Safety

The package is designed to be thread-safe and can handle concurrent requests across
multiple goroutines and nodes. All state modifications are protected by appropriate
mutex locks.

# Error Handling

The service handles various error conditions:
  - Invalid configurations (negative limits, zero durations)
  - Network partitions between nodes
  - Node failures and cluster changes
  - Race conditions in distributed state

See the RatelimitRequest and RatelimitResponse types for detailed documentation
of the API contract and error conditions.
*/
package ratelimit
