package ratelimit

import (
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
)

type Service struct {
	logger      logging.Logger
	ratelimiter ratelimit.Ratelimiter
}

type Config struct {
	Logger logging.Logger
}

func New(cfg Config) (*Service, error) {

	return &Service{
		logger:      cfg.Logger,
		ratelimiter: ratelimit.NewInMemory(),
	}, nil
}
