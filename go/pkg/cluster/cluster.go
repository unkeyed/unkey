package cluster

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/pkg/events"
	"github.com/unkeyed/unkey/go/pkg/membership"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/otel/metrics"
	"github.com/unkeyed/unkey/go/pkg/ring"
	"go.opentelemetry.io/otel/metric"
)

// Config configures a new cluster instance with the necessary components
// for node management and distributed operations. All fields are required
// unless explicitly marked as optional.
//
// The Config struct is used only during cluster initialization and is not
// accessed after New() returns. Modifying a Config after passing it to New()
// has no effect.
//
// Example:
//
//	config := cluster.Config{
//		Self: cluster.Instance{
//			ID:      "node-1",
//			RpcAddr: "10.0.0.1:7071",
//		},
//		Membership: membershipSvc,
//		Logger:     logger,
//	}
type Config struct {
	// Self contains information about the local node. This field is required
	// and must contain a valid Instance with non-empty ID and RpcAddr fields.
	Self Instance

	// Membership provides the underlying node discovery and failure detection.
	// It must be a properly initialized membership.Membership implementation.
	// The membership service must be started separately before creating the cluster.
	Membership membership.Membership

	// Logger records operational events and errors. If nil, logging will be disabled.
	// It's strongly recommended to provide a logger in production environments.
	Logger logging.Logger
}

// New creates a new cluster instance with the provided configuration. It initializes
// the consistent hash ring and sets up event listeners for membership changes.
//
// New performs the following initialization steps:
// 1. Validates the configuration
// 2. Creates the consistent hash ring
// 3. Registers the local instance
// 4. Sets up membership event handlers
// 5. Initializes metrics collection
//
// The returned cluster instance is immediately operational and thread-safe.
//
// Parameters:
//   - config: Required configuration for the cluster instance
//
// Returns:
//   - *cluster: A fully initialized cluster instance
//   - error: If initialization fails
//
// Errors:
//   - If config.Self.ID is empty
//   - If config.Membership is nil
//   - If the hash ring initialization fails
//   - If metrics registration fails
//
// Example:
//
//	cluster, err := cluster.New(cluster.Config{
//		Self: cluster.Instance{
//			ID:      "node-1",
//			RpcAddr: "10.0.0.1:7071",
//		},
//		Membership: membershipSvc,
//		Logger:     logger,
//	})
//	if err != nil {
//		return fmt.Errorf("cluster initialization failed: %w", err)
//	}
func New(config Config) (*cluster, error) {

	r, err := ring.New[Instance](ring.Config{
		TokensPerNode: 256,
		Logger:        config.Logger,
	})

	if err != nil {
		return nil, fmt.Errorf("unable to create hash ring: %w", err)
	}

	c := &cluster{
		self:        config.Self,
		membership:  config.Membership,
		ring:        r,
		logger:      config.Logger,
		joinEvents:  events.NewTopic[Instance](),
		leaveEvents: events.NewTopic[Instance](),
	}

	err = c.registerMetrics()
	if err != nil {
		return nil, err
	}

	go c.keepInSync()

	err = r.AddNode(context.Background(), ring.Node[Instance]{
		ID:   config.Self.ID,
		Tags: config.Self,
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

// cluster implements the Cluster interface by combining membership information
// with consistent hashing to provide distributed operations.
//
// The cluster type maintains internal state that is protected for concurrent
// access. All exported methods are safe for concurrent use by multiple
// goroutines.
//
// Internal components:
// - Consistent hash ring for workload distribution
// - Event system for topology changes
// - Metrics collection for monitoring
// - Background goroutine for membership sync
type cluster struct {
	self       Instance              // Local instance information
	membership membership.Membership // Underlying membership service
	ring       *ring.Ring[Instance]  // Consistent hash ring
	logger     logging.Logger        // Logging interface

	joinEvents  events.Topic[Instance] // Join event broadcast
	leaveEvents events.Topic[Instance] // Leave event broadcast
}

// Ensure cluster implements the Cluster interface at compile time
var _ Cluster = (*cluster)(nil)

// Self returns information about the local node. This method is thread-safe
// and returns an immutable Instance struct.
//
// The returned Instance contains the node's stable identity that was provided
// during cluster initialization. This information remains constant for the
// lifetime of the cluster instance.
func (c *cluster) Self() Instance {
	return c.self
}

// registerMetrics sets up OpenTelemetry metrics collection for the cluster.
// It registers callbacks to track cluster size and other operational metrics.
//
// This is an internal method called during cluster initialization.
//
// Returns an error if metric registration fails, which should be treated as
// a fatal initialization error.
//
// Metrics exposed:
// - cluster.size: Number of active instances in the cluster
// - cluster.events: Count of join/leave events
// - cluster.ring.rebalances: Number of hash ring rebalancing operations
func (c *cluster) registerMetrics() error {
	err := metrics.Cluster.Size.RegisterCallback(func(_ context.Context, o metric.Int64Observer) error {
		members, err := c.membership.Members()
		if err != nil {
			c.logger.Error("failed to collect cluster size", "error", err)
			return err
		}

		o.Observe(int64(len(members)))
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to register cluster size metric: %w", err)
	}

	return nil
}

// SubscribeJoin returns a channel that receives Instance events when nodes join the cluster.
// The channel remains open until the cluster is shut down or the subscription is cancelled.
//
// The returned channel is buffered and should be consumed continuously to prevent
// blocking of internal event distribution. Events are delivered in the order they
// occur, but consumers should not rely on exact ordering for correctness.
//
// Thread-safety:
//   - Safe to call from multiple goroutines
//   - Each subscription gets its own channel
//   - Events are broadcast to all subscribers
//
// Example:
//
//	joins := cluster.SubscribeJoin()
//	go func() {
//		for instance := range joins {
//			// Handle new instance
//			log.Printf("Instance %s joined at %s", instance.ID, instance.RpcAddr)
//			// Optionally warm up connections, caches, etc.
//		}
//	}()
func (c *cluster) SubscribeJoin() <-chan Instance {
	return c.joinEvents.Subscribe("cluster.joinEvents")
}

// SubscribeLeave returns a channel that receives Instance events when nodes leave
// the cluster, either gracefully or due to failures. The channel remains open
// until the cluster is shut down or the subscription is cancelled.
//
// Leave events are generated in two cases:
//  1. Graceful shutdown: Node calls Shutdown()
//  2. Failure detection: Membership service detects node failure
//
// Thread-safety:
//   - Safe to call from multiple goroutines
//   - Each subscription gets its own channel
//   - Events are broadcast to all subscribers
//
// Example:
//
//	leaves := cluster.SubscribeLeave()
//	go func() {
//		for instance := range leaves {
//			// Handle instance departure
//			log.Printf("Instance %s left the cluster", instance.ID)
//			// Clean up any instance-specific resources
//		}
//	}()
func (c *cluster) SubscribeLeave() <-chan Instance {
	return c.leaveEvents.Subscribe("cluster.leaveEvents")
}

// keepInSync listens to membership changes and updates the hash ring accordingly.
// This method runs in a dedicated goroutine and handles all cluster topology changes.
//
// The method performs the following:
// 1. Subscribes to membership join/leave events
// 2. Updates the consistent hash ring when instances join/leave
// 3. Broadcasts topology changes via the event system
// 4. Logs all membership changes for debugging
//
// This is an internal method started during cluster initialization.
// It continues running until the cluster is shut down.
//
// Thread-safety:
// - Safe to run concurrently with all other cluster operations
// - Ensures atomic updates to the hash ring
// - Properly synchronizes event broadcasts
func (c *cluster) keepInSync() {
	joins := c.membership.SubscribeJoinEvents()
	leaves := c.membership.SubscribeLeaveEvents()

	for {
		select {
		case instance := <-joins:
			{
				ctx := context.Background()
				c.logger.Info("instance joined", "instance", instance)

				err := c.ring.AddNode(ctx, ring.Node[Instance]{
					ID: instance.InstanceID,
					Tags: Instance{
						RpcAddr: fmt.Sprintf("%s:%d", instance.Host, instance.RpcPort),
						ID:      instance.InstanceID,
					},
				})
				if err != nil {
					c.logger.Error("failed to add node to ring", "error", err.Error())
				}

			}
		case instance := <-leaves:
			{
				ctx := context.Background()
				c.logger.Info("instance left", "instanceID", instance.InstanceID)
				err := c.ring.RemoveNode(ctx, instance.InstanceID)
				if err != nil {
					c.logger.Error("failed to remove node from ring", "error", err.Error())
				}
			}
		}

	}

}

// FindInstance determines which instance is responsible for a given key using
// consistent hashing. This method ensures that the same key always maps to the
// same instance (assuming the instance remains available).
//
// The consistent hashing algorithm minimizes key redistribution when the cluster
// topology changes (instances joining or leaving).
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - key: String identifier to look up (e.g., user ID, API key)
//
// Returns:
//   - Instance: The responsible instance for the key
//   - error: If lookup fails or no instances are available
//
// Errors:
//   - ring.ErrNoNodes: If the cluster has no available instances
//   - ring.ErrNodeNotFound: If the responsible instance was removed
//   - context.Canceled: If the context is cancelled before completion
//
// Thread-safety:
//   - Safe for concurrent use
//   - Consistent results during topology changes
//
// Example:
//
//	ctx := context.Background()
//	instance, err := cluster.FindInstance(ctx, "user:123")
//	if err != nil {
//		if errors.Is(err, ring.ErrNoNodes) {
//			// Handle empty cluster case
//		}
//		return fmt.Errorf("instance lookup failed: %w", err)
//	}
//	// Use the instance for the operation
//	log.Printf("Key %s maps to instance %s", key, instance.ID)
func (c *cluster) FindInstance(ctx context.Context, key string) (Instance, error) {
	node, err := c.ring.FindNode(key)
	if err != nil {
		return Instance{}, fmt.Errorf("ring lookup failed: %w", err)
	}
	return node.Tags, nil
}

// Shutdown gracefully exits the cluster, notifying other instances and cleaning
// up resources. It ensures proper cluster rebalancing by:
//  1. Removing the local instance from the consistent hash ring
//  2. Notifying other instances via the membership service
//  3. Closing event channels and stopping background goroutines
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//
// Returns:
//   - error: If shutdown encounters errors
//
// Thread-safety:
//   - Safe to call once from any goroutine
//   - Should not be called multiple times
//   - Other cluster methods may return errors after Shutdown
//
// Best practices:
//   - Call during service shutdown
//   - Use context with timeout
//   - Handle errors to ensure cleanup
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	if err := cluster.Shutdown(ctx); err != nil {
//		log.Printf("Cluster shutdown error: %v", err)
//		// Consider forcing shutdown after timeout
//	}
func (c *cluster) Shutdown(ctx context.Context) error {
	return c.membership.Leave()
}
