package ratelimit

import (
	"time"

	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/batch"
	"github.com/unkeyed/unkey/apps/agent/pkg/cluster"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/repeat"
)

type service struct {
	logger      logging.Logger
	ratelimiter ratelimit.Ratelimiter
	cluster     cluster.Cluster

	batcher *batch.BatchProcessor[*ratelimitv1.PushPullEvent]
	metrics *metrics
}

type Config struct {
	Logger  logging.Logger
	Cluster cluster.Cluster
}

func New(cfg Config) (Service, error) {
	s := &service{
		logger:      cfg.Logger,
		ratelimiter: ratelimit.NewFixedWindow(cfg.Logger.With().Str("ratelimiter", "fixedWindow").Logger()),
		cluster:     cfg.Cluster,
		metrics:     newMetrics(),
	}

	if cfg.Cluster != nil {

		s.batcher = batch.New(batch.Config[*ratelimitv1.PushPullEvent]{
			BatchSize:     50,
			BufferSize:    100000,
			FlushInterval: time.Millisecond * 100,
			Flush:         s.flushPushPull,
		})

		repeat.Every(time.Minute, func() {
			s.logger.Info().Int("size", s.batcher.Size()).Msg("pushPull backlog")
		})

	}

	repeat.Every(time.Minute, func() {
		m := s.Metrics()
		for key, peers := range m {
			if len(peers) > 0 {
				// Our hashring ensures that a single key is only ever sent to a single node for pushpull
				// In theory at least..
				s.logger.Warn().Str("key", key).Interface("peers", peers).Msg("ratelimit had multiple origins")
			}

		}
	})

	return s, nil
}

func (s *service) Metrics() map[string]map[string]int {
	return s.metrics.ReadAndReset()
}
