package caches

import (
	"time"

	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Caches holds all shared cache instances for the ctrl application.
type Caches struct {
	Domains cache.Cache[string, db.CustomDomain]
}

type Config struct {
	Logger logging.Logger
	Clock  clock.Clock
}

func New(cfg Config) (*Caches, error) {
	clk := cfg.Clock
	if clk == nil {
		clk = clock.New()
	}

	domains, err := cache.New(cache.Config[string, db.CustomDomain]{
		Fresh:    5 * time.Minute,
		Stale:    10 * time.Minute,
		MaxSize:  10000,
		Logger:   cfg.Logger,
		Resource: "domains",
		Clock:    clk,
	})
	if err != nil {
		return nil, err
	}

	return &Caches{
		Domains: domains,
	}, nil
}
