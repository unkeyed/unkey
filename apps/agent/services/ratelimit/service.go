package ratelimit

import (
	"github.com/unkeyed/unkey/apps/agent/pkg/cluster"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
)

type service struct {
	logger      logging.Logger
	ratelimiter ratelimit.Ratelimiter
	cluster cluster.Cluster
}

type Config struct {
	Logger logging.Logger
}

func New(cfg Config) (Service, error) {

	return &service{
		logger:      cfg.Logger,
		ratelimiter: ratelimit.NewFixedWindow(),
	}, nil
}
