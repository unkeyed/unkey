package ratelimit

import (
	"time"

	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/batch"
	"github.com/unkeyed/unkey/apps/agent/pkg/cluster"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/repeat"
)

type service struct {
	logger      logging.Logger
	ratelimiter ratelimit.Ratelimiter
	cluster     cluster.Cluster

	batcher            *batch.BatchProcessor[*ratelimitv1.PushPullEvent]
	metrics            metrics.Metrics
	consistencyChecker *consistencyChecker
}

type Config struct {
	Logger  logging.Logger
	Metrics metrics.Metrics
	Cluster cluster.Cluster
}

func New(cfg Config) (Service, error) {
	s := &service{
		logger:             cfg.Logger,
		ratelimiter:        ratelimit.NewFixedWindow(cfg.Logger.With().Str("ratelimiter", "fixedWindow").Logger()),
		cluster:            cfg.Cluster,
		metrics:            cfg.Metrics,
		consistencyChecker: newConsistencyChecker(cfg.Logger),
	}

	if cfg.Cluster != nil {

		maxBufferSize := 100000
		s.batcher = batch.New(batch.Config[*ratelimitv1.PushPullEvent]{
			BatchSize:     50,
			BufferSize:    maxBufferSize,
			FlushInterval: time.Millisecond * 100,
			Flush:         s.flushPushPull,
		})

		repeat.Every(time.Minute, func() {
			s.metrics.Record(metrics.ChannelBuffer{
				ID:      "pushpull",
				Size:    s.batcher.Size(),
				MaxSize: maxBufferSize,
			})
		})

	}

	return s, nil
}
