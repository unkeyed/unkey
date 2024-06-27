package ratelimit

import (
	"github.com/unkeyed/unkey/apps/agent/pkg/cluster"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
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
	tracer      tracing.Tracer
	ratelimiter ratelimit.Ratelimiter
	cluster     cluster.Cluster

	pushPullC chan pushPullEvent
}

type Config struct {
	Logger  logging.Logger
	Tracer  tracing.Tracer
	Cluster cluster.Cluster
}

func New(cfg Config) (Service, error) {
	s := &service{
		logger:      cfg.Logger,
		tracer:      cfg.Tracer,
		ratelimiter: ratelimit.NewFixedWindow(cfg.Logger.With().Str("ratelimiter", "fixedWindow").Logger()),
		cluster:     cfg.Cluster,
	}

	if cfg.Cluster != nil {
		s.pushPullC = make(chan pushPullEvent)
		go s.runPushPullSync()
	}
	return s, nil
}
