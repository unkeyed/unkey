package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/unkeyed/unkey/go/pkg/logging"
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
		return nil, fmt.Errorf("failed to advertise state to redis")
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

func (r *Redis) key() string {
	return fmt.Sprintf("discovery::nodes::%s", r.nodeID)
}
func (r *Redis) advertise(ctx context.Context) error {
	return r.rdb.Set(ctx, r.key(), r.addr, r.ttl).Err()
}

func (r *Redis) Discover() ([]string, error) {
	pattern := r.key()
	pattern = strings.ReplaceAll(pattern, r.nodeID, "*")
	keys, err := r.rdb.Keys(context.Background(), pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %w", err)
	}

	if len(keys) == 0 {
		return []string{}, nil
	}

	results, err := r.rdb.MGet(context.Background(), keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get addresses: %w", err)
	}

	addrs := make([]string, len(results))
	var ok bool
	for i, addr := range results {
		addrs[i], ok = addr.(string)
		if !ok {
			return nil, fmt.Errorf("invalid address type")
		}
	}

	return addrs, nil
}

func (r *Redis) Shutdown(ctx context.Context) error {
	r.shutdownC <- struct{}{}
	return r.rdb.Del(ctx, r.key()).Err()
}
