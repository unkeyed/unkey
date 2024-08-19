package ratelimit

import (
	"sync"
	"time"

	"github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1/ratelimitv1connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/cluster"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/pkg/prometheus"
	"github.com/unkeyed/unkey/apps/agent/pkg/repeat"
)

type service struct {
	logger      logging.Logger
	ratelimiter *slidingWindow
	cluster     cluster.Cluster

	syncBuffer         chan syncWithOriginRequest
	metrics            metrics.Metrics
	consistencyChecker *consistencyChecker

	peersMu sync.RWMutex
	// url -> client map
	peers map[string]ratelimitv1connect.RatelimitServiceClient

	shutdownCh chan struct{}
}

type Config struct {
	Logger  logging.Logger
	Metrics metrics.Metrics
	Cluster cluster.Cluster
}

func New(cfg Config) (Service, error) {

	s := &service{
		logger:             cfg.Logger,
		ratelimiter:        NewSlidingWindow(cfg.Logger.With().Str("ratelimiter", "slidingWindow").Logger()),
		cluster:            cfg.Cluster,
		metrics:            cfg.Metrics,
		consistencyChecker: newConsistencyChecker(cfg.Logger),
		peersMu:            sync.RWMutex{},
		peers:              map[string]ratelimitv1connect.RatelimitServiceClient{},
		// Only set if we have a cluster
		syncBuffer: nil,
		shutdownCh: make(chan struct{}),
	}

	if cfg.Cluster != nil {

		s.syncBuffer = make(chan syncWithOriginRequest, 10000)
		// Process the individual requests to the origin and update local state
		// We're using 32 goroutines to parallelise the network requests'
		s.logger.Info().Msg("starting syncWithOrigin background jobs")
		for range 32 {
			go func() {
				for {
					select {
					case <-s.shutdownCh:
						return
					case req := <-s.syncBuffer:
						s.syncWithOrigin(req)
					}
				}
			}()
		}

		repeat.Every(time.Second, func() {

			prometheus.ChannelBuffer.With(map[string]string{
				"id": "pushpull.syncWithOrigin",
			}).Set(float64(len(s.syncBuffer)) / float64(cap(s.syncBuffer)))

		})

	}

	return s, nil
}
