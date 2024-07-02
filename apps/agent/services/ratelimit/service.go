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
	}

	if cfg.Cluster != nil {

		s.batcher = batch.New(batch.Config[*ratelimitv1.PushPullEvent]{
			BatchSize:     1000,
			BufferSize:    100000,
			FlushInterval: time.Millisecond * 200,
			Flush:         s.flushPushPull,
		})

		repeat.Every(time.Minute, func() {
			s.logger.Info().Int("size", s.batcher.Size()).Msg("pushPull backlog")
		})

	}
	return s, nil
}
