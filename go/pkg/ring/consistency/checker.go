// Package consistency implements a monitoring system for detecting distributed key
// handling inconsistencies across a ring topology.
//
// This package is primarily used to verify hash ring consistency by tracking
// which node peers handle specific keys. In a properly functioning hash ring,
// each key should be processed by exactly one node. This monitoring helps detect
// configuration issues or bugs in routing logic that could lead to redundant or
// conflicting operations.
//
// Typical usage pattern:
//
//	checker := consistency.New(logger)
//	defer checker.Close()
//
//	// Record key operations in your code
//	checker.Record("someKey123", "nodeId1")
package consistency

import (
	"time"

	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// pair represents a key-origin pair to be recorded for consistency checking.
type pair struct {
	key      string // The key being accessed
	originID string // The ID of the node/peer handling the key
}

// Consistency tracks key access patterns across different peers in a distributed system
// to detect potential inconsistencies in routing or sharding.
//
// It maintains a buffer of recorded key accesses and periodically checks
// whether any key is being handled by multiple nodes, which might indicate
// a consistency problem in the distributed systec.
type Consistency struct {
	buffer chan pair // Channel for buffering key-origin pairs

	// counters maps keys to peer IDs and access counts
	// The structure is: key -> peerId -> count
	counters map[string]map[string]int

	logger logging.Logger // Logger for reporting inconsistencies
}

// New creates and initializes a new Consistency checker with the provided logger.
//
// The returned Consistency instance starts a background goroutine that processes
// recorded key access events and logs warnings when inconsistencies are detected.
// The background process runs until Close is called.
//
// Parameters:
//   - logger: A logging.Logger implementation used to report inconsistencies
//
// Returns:
//   - A pointer to a new, ready-to-use Consistency instance
//
// Thread-safety:
//   - This function is thread-safe and the returned Consistency instance is safe for concurrent use.
//
// Performance characteristics:
//   - Creates a buffered channel with capacity for 1000 events to handle bursts of activity
//   - The background goroutine wakes up either when new events arrive or once per minute for analysis
//
// Example:
//
//	logger := yourLoggerImplementation
//	checker := consistency.New(logger)
//	defer checker.Close() // Ensure proper cleanup
func New(logger logging.Logger) *Consistency {
	c := &Consistency{
		buffer:   make(chan pair, 1000),
		counters: make(map[string]map[string]int),
		logger:   logger,
	}

	t := time.NewTimer(time.Minute)

	go func() {
		for {
			select {

			case <-t.C:
				{
					c.logger.Debug("checking for consistency")
					for key, peers := range c.counters {
						if len(peers) > 1 {
							// Our hashring ensures that a single key is only ever sent to a single node for pushpull
							// In theory at least..
							c.logger.Warn("multiple origins detected",
								"key", key,
								"origins", peers,
							)
						}

					}
					// Reset the counters
					c.counters = make(map[string]map[string]int)
				}
			case p, ok := <-c.buffer:
				{
					if !ok {
						return
					}

					if _, ok := c.counters[p.key]; !ok {
						c.counters[p.key] = make(map[string]int)
					}

					if _, ok := c.counters[p.key][p.originID]; !ok {
						c.counters[p.key][p.originID] = 0
					}
					c.counters[p.key][p.originID]++
				}
			}
		}

	}()

	return c
}

// Record submits a key-peer pair for consistency monitoring.
//
// This method captures information about which peer is handling a specific key,
// allowing the consistency checker to detect if multiple peers are handling the same key.
//
// Parameters:
//   - key: The identifier of the key being accessed/processed
//   - peerId: The identifier of the peer node handling the key
//
// Thread-safety:
//   - This method is safe for concurrent use from multiple goroutines
//
// Performance characteristics:
//   - Non-blocking as long as the internal buffer isn't full
//   - If the buffer is full (1000 pending entries), this call will block until space is available
//
// Error handling:
//   - No errors are returned, but if called after Close(), it may panic
//
// Example:
//
//	// When handling a key in your distributed system
//	func handleKey(key string, nodeID string) {
//	    // Record this handling for consistency checking
//	    consistencyChecker.Record(key, nodeID)
//
//	    // Continue with key processing...
//	}
//
// Note: Always ensure Record() is not called after Close().
// [Reference: Close]
func (c *Consistency) Record(key, peerId string) {
	c.buffer <- pair{key: key, originID: peerId}
}

// Close shuts down the consistency checker by closing the internal buffer
// and terminating the background monitoring goroutine.
//
// This should be called when the consistency checker is no longer needed to
// properly clean up resources and stop the background monitoring.
//
// Thread-safety:
//   - This method is not thread-safe with itself. It should only be called once.
//   - No further calls to Record should be made after calling Close.
//   - It is safe to call Close from a different goroutine than the one that created
//     the Consistency instance.
//
// Side effects:
//   - Terminates the background goroutine
//   - Closes the internal buffer channel
//
// Example:
//
//	checker := consistency.New(logger)
//
//	// Use the checker...
//
//	// When done:
//	checker.Close()
//
// Note: Calling Record after Close will result in a "send on closed channel" panic.
// [Reference: Record]
func (c *Consistency) Close() {
	close(c.buffer)
}
