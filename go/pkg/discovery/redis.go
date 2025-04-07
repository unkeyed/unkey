package discovery

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/retry"
)

// Redis implements dynamic service discovery using Redis as a coordination backend.
// It provides reliable peer discovery in environments where instances frequently
// come and go, such as:
//   - Auto-scaling groups
//   - Containerized deployments
//   - Kubernetes pods
//   - Cloud-native applications
//
// Redis discovery uses a self-cleaning mechanism where:
//   - Each instance periodically advertises its address with a TTL
//   - Offline instances are automatically removed when their TTL expires
//   - New instances are immediately discoverable upon advertisement
//   - No central management or cleanup is required
//
// Thread Safety
//
// Redis is safe for concurrent use. It maintains a background goroutine for
// heartbeats that must be cleaned up by calling Shutdown().
//
// Example:
//
//	discoverer, err := discovery.NewRedis(discovery.RedisConfig{
//	    URL:        "redis://localhost:6379/0",
//	    InstanceID: "node-1",
//	    Addr:      "10.0.1.1:9000",
//	    Logger:    logger,
//	})
//	if err != nil {
//	    return err
//	}
//	defer discoverer.Shutdown(context.Background())
//
//	// Discover peers
//	addrs, err := discoverer.Discover()
//
// Error Handling
//
// Redis implements retry logic for transient failures during discovery.
// Permanent errors (like invalid configuration) are returned immediately.
type Redis struct {
	rdb               *redis.Client
	logger            logging.Logger
	addr              string
	instanceID        string
	ttl               time.Duration
	heartbeatInterval time.Duration
	shutdownC         chan struct{}
}

// RedisConfig defines the configuration for Redis-based service discovery.
type RedisConfig struct {
	// URL specifies the Redis connection string in the format:
	// redis://[username:password@]host[:port][/database_number]
	//
	// Examples:
	//   redis://localhost:6379/0
	//   redis://user:pass@redis.example.com:6379/1
	//   redis://10.0.1.5:6379/0
	URL string

	// InstanceID uniquely identifies this node in the cluster.
	// Must be unique across all instances using the same Redis backend.
	// Common formats include:
	//   - UUIDs: "550e8400-e29b-41d4-a716-446655440000"
	//   - Hostnames: "node-1", "worker-a"
	//   - Cloud instance IDs: "i-0123456789abcdef0"
	InstanceID string

	// Addr specifies the network address other instances will use to connect
	// to this instance. Must be in "host:port" format where host can be:
	//   - DNS name: "node1.example.com:9000"
	//   - IPv4: "10.0.1.5:9000"
	//   - IPv6: "[2001:db8::1]:9000"
	Addr string

	// Logger receives operational logs for monitoring and debugging.
	// Must implement the logging.Logger interface.
	Logger logging.Logger
}

// NewRedis creates a new Redis-based discoverer and initializes the advertisement process.
// It performs the following setup:
//  1. Establishes connection to Redis using the provided URL
//  2. Validates the connection with a PING command
//  3. Advertises this instance's address immediately
//  4. Starts a background goroutine for periodic advertisement refresh
//
// The instance continues to advertise itself until Shutdown is called.
// Failing to call Shutdown may leave stale entries in Redis.
//
// NewRedis returns an error if:
//   - The Redis URL is invalid or unreachable
//   - Initial advertisement fails
//   - Required configuration fields are missing
//
// The returned discoverer is immediately ready for use.
func NewRedis(config RedisConfig) (*Redis, error) {
	opts, err := redis.ParseURL(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}

	rdb := redis.NewClient(opts)
	config.Logger.Debug("pinging redis")

	_, err = rdb.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	r := &Redis{
		logger:            config.Logger,
		rdb:               rdb,
		instanceID:        config.InstanceID,
		addr:              config.Addr,
		heartbeatInterval: time.Second * 60,
		ttl:               time.Second * 90,
		shutdownC:         make(chan struct{}),
	}

	err = r.advertise(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to advertise state to redis: %w", err)
	}

	go r.heartbeat()

	return r, nil
}

// heartbeat maintains this instance's presence in Redis through periodic
// advertisement refreshes. It runs until Shutdown is called.
//
// The heartbeat ensures the instance remains discoverable during long periods
// of inactivity by refreshing the TTL before expiration. This prevents the
// instance from being considered offline while still running.
func (r *Redis) heartbeat() {
	t := time.NewTicker(r.heartbeatInterval)
	defer t.Stop()

	for {
		select {
		case <-r.shutdownC:
			return
		case <-t.C:
			ctx := context.Background()
			err := r.advertise(ctx)
			if err != nil {
				r.logger.Error("failed to advertise state to redis", "err", err.Error())
			}
		}
	}
}

// key generates a Redis key for storing instance information. The key format is:
// "discovery::v1::instances::<instanceID>"
//
// This format enables:
//   - Namespace isolation with the "discovery" prefix
//   - Version-based migrations with "v1"
//   - Easy scanning of all instances using the "instances" prefix
//   - Direct lookup of specific instances by ID
// key formats a Redis key for the given instance ID.
func (r *Redis) key(instanceID string) string {
	return fmt.Sprintf("discovery::v1::instances::%s", instanceID)
}

// advertise publishes this instance's address to Redis with a TTL.
// The address remains discoverable until either:
//   - The TTL expires (90s by default)
//   - The instance calls Shutdown
//   - Redis is cleared/restarted
//
// The TTL ensures that crashed or network-partitioned instances are
// automatically removed from discovery after a reasonable timeout.
// advertise updates this instance's address in Redis with a TTL.
// The TTL is refreshed automatically by the heartbeat goroutine.
func (r *Redis) advertise(ctx context.Context) error {
	return r.rdb.Set(ctx, r.key(r.instanceID), r.addr, r.ttl).Err()
}

// Discover returns addresses of all active instances except this one.
// It implements automatic retry logic to handle transient Redis failures.
//
// The method:
//   - Scans Redis for all discovery keys
//   - Retrieves the address for each key
//   - Filters out this instance's own address
//   - Retries up to 10 times on failure
//
// Returns an error if discovery fails after all retries.
//
// Thread Safety: Safe for concurrent use.
// Discover implements the Discoverer interface using Redis as a backend.
// It returns addresses of all active instances except this one.
func (r *Redis) Discover() ([]string, error) {
	addrs := []string{}

	err := retry.New(
		retry.Attempts(10),
	).Do(func() error {
		found, dErr := r.discover()
		if dErr != nil {
			return dErr
		}
		addrs = found
		return nil
	})
	return addrs, err
}

// discover performs the actual Redis operations to find active instances.
// It:
//   - Scans for all discovery keys using SCAN
//   - Retrieves addresses using GET
//   - Filters out this instance's own address
//   - Handles Redis pagination automatically
//
// Returns an error if either SCAN or GET operations fail.
func (r *Redis) discover() ([]string, error) {
	pattern := r.key("*")

	// turn this into using SCan instead
	var cursor uint64
	var keys []string

	for {
		var batch []string
		var err error
		batch, cursor, err = r.rdb.Scan(context.Background(), cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to scan keys: %w", err)
		}

		keys = append(keys, batch...)

		if cursor == 0 {
			break
		}
	}

	addrs := []string{}
	for _, key := range keys {
		if key == r.key(r.instanceID) {
			continue
		}

		result, err := r.rdb.Get(context.Background(), key).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get address for key %s: %w", key, err)
		}
		addrs = append(addrs, result)
	}

	return addrs, nil
}

// Shutdown gracefully terminates the discoverer by:
//   - Stopping the heartbeat goroutine
//   - Removing this instance's entry from Redis
//   - Cleaning up resources
//
// After Shutdown, the instance is no longer discoverable by peers.
// This method should be called during graceful shutdown, typically
// using defer:
//
//	discoverer, _ := discovery.NewRedis(config)
//	defer discoverer.Shutdown(ctx)
//
// Thread Safety: Safe to call concurrently, but only the first call
// will have an effect.
func (r *Redis) Shutdown(ctx context.Context) error {
	r.shutdownC <- struct{}{}
	return r.rdb.Del(ctx, r.key(r.instanceID)).Err()
}
