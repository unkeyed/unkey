package ratelimit

import (
	"github.com/unkeyed/unkey/apps/agent/pkg/cluster"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
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
		ratelimiter: ratelimit.NewFixedWindow(),
		cluster:     cfg.Cluster,
	}

	if cfg.Cluster != nil {
		s.pushPullC = make(chan pushPullEvent)
		go s.runPushPullSync()
	}
	return s, nil
}
