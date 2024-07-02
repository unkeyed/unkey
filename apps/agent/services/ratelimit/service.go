package ratelimit

import (
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/cluster"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/repeat"
)

type pushPullEvent struct {
	identifier string
	limit      int64
	// milliseconds
	duration int64
	cost     int64
}

type service struct {
	logger      logging.Logger
	ratelimiter ratelimit.Ratelimiter
	cluster     cluster.Cluster

	pushPullC chan pushPullEvent
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
		s.pushPullC = make(chan pushPullEvent, 100000)
		go s.runPushPullSync()

		repeat.Every(time.Minute, func() {
			s.logger.Info().Int("size", len(s.pushPullC)).Msg("pushPull backlog")
		})

	}
	return s, nil
}
