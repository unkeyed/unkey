package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/unkeyed/unkey/go/pkg/logging"
	"github.com/unkeyed/unkey/go/pkg/retry"
)

type Redis struct {
	rdb    *redis.Client
	logger logging.Logger

	addr   string
	nodeID string

	ttl               time.Duration
	heartbeatInterval time.Duration
	shutdownC         chan struct{}
}

type RedisConfig struct {
	URL    string
	NodeID string
	Addr   string
	Logger logging.Logger
}

func NewRedis(config RedisConfig) (*Redis, error) {
	opts, err := redis.ParseURL(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}

	rdb := redis.NewClient(opts)
	config.Logger.Info(context.Background(), "pinging redis", slog.Any("opts", opts))

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
				r.logger.Error(ctx, "failed to advertise state to redis", slog.String("err", err.Error()))
			}
		}
	}
}

func (r *Redis) key(nodeID string) string {
	return fmt.Sprintf("discovery::nodes::%s", nodeID)
}
func (r *Redis) advertise(ctx context.Context) error {
	return r.rdb.Set(ctx, r.key(r.nodeID), r.addr, r.ttl).Err()
}

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

func (r *Redis) Shutdown(ctx context.Context) error {
	r.shutdownC <- struct{}{}
	return r.rdb.Del(ctx, r.key(r.nodeID)).Err()
}
