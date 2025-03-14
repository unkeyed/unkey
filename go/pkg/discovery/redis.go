package discovery

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/retry"
)

// Redis implements the Discoverer interface using a Redis backend for service discovery.
// This discovery method is suitable for dynamic environments where nodes may come and
// go frequently, such as auto-scaling groups or containerized deployments.
//
// Redis discovery works by:
// 1. Having each node periodically advertise itself by writing its address to Redis
// 2. Setting a TTL on each advertisement so that offline nodes are automatically removed
// 3. Providing a mechanism to enumerate all currently active nodes
//
// This approach provides self-cleaning discovery without any centralized management.
type Redis struct {
	rdb    *redis.Client
	logger logging.Logger

	addr   string
	nodeID string

	ttl               time.Duration
	heartbeatInterval time.Duration
	shutdownC         chan struct{}
}

// RedisConfig defines the configuration options for Redis-based discovery.
type RedisConfig struct {
	// URL is the Redis connection string, e.g., "redis://user:pass@localhost:6379/0"
	URL string

	// NodeID is the unique identifier for this node
	NodeID string

	// Addr is the address other nodes should use to connect to this node
	// This is the address that will be advertised in Redis
	Addr string

	// Logger is used for operational logging
	Logger logging.Logger
}

// NewRedis creates a new Redis-based discoverer and starts the advertisement process.
// The returned discoverer will automatically:
// 1. Connect to the specified Redis instance
// 2. Immediately advertise this node's address
// 3. Start a background goroutine to periodically refresh the advertisement
//
// The node will continue to advertise itself until Shutdown is called.
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
		nodeID:            config.NodeID,
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

// heartbeat periodically refreshes this node's advertisement in Redis.
// This ensures that the node remains discoverable even during long periods
// of inactivity. It also renews the TTL, preventing the node from being
// considered offline if it's still running.
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

// key generates the Redis key used to store node information.
// The key includes a common prefix for all discovery entries and the node ID,
// making it easy to scan for all nodes while also retrieving specific nodes.
func (r *Redis) key(nodeID string) string {
	return fmt.Sprintf("discovery::nodes::%s", nodeID)
}

// advertise publishes this node's address to Redis with a TTL.
// Other nodes can discover this node by scanning for keys with the discovery prefix.
func (r *Redis) advertise(ctx context.Context) error {
	return r.rdb.Set(ctx, r.key(r.nodeID), r.addr, r.ttl).Err()
}

// Discover returns a list of addresses for all active nodes registered in Redis.
// It scans for all keys with the discovery prefix and retrieves their values,
// which contain the addresses of the nodes.
//
// This implementation includes retry logic to handle transient Redis failures,
// making the discovery process more resilient in production environments.
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

// discover performs the actual Redis query to find all active nodes.
// It scans for all keys with the discovery prefix and retrieves their values.
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
		result, err := r.rdb.Get(context.Background(), key).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get address for key %s: %w", key, err)
		}
		addrs = append(addrs, result)
	}

	return addrs, nil
}

// Shutdown stops the heartbeat goroutine and removes this node's entry from Redis.
// This should be called during graceful shutdown to ensure the node is no longer
// discoverable once it's offline.
func (r *Redis) Shutdown(ctx context.Context) error {
	r.shutdownC <- struct{}{}
	return r.rdb.Del(ctx, r.key(r.nodeID)).Err()
}
