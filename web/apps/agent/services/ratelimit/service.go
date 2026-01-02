package ratelimit

import (
	"sync"
	"time"

	"connectrpc.com/connect"

	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1/ratelimitv1connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/apps/agent/pkg/cluster"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/pkg/prometheus"
	"github.com/unkeyed/unkey/apps/agent/pkg/repeat"
)

type service struct {
	logger  logging.Logger
	cluster cluster.Cluster

	mitigateBuffer     chan mitigateWindowRequest
	syncBuffer         chan syncWithOriginRequest
	metrics            metrics.Metrics
	consistencyChecker *consistencyChecker

	peersMu sync.RWMutex
	// url -> client map
	peers map[string]ratelimitv1connect.RatelimitServiceClient

	shutdownCh chan struct{}

	bucketsMu sync.RWMutex
	// identifier+sequence -> bucket
	buckets             map[string]*bucket
	leaseIdToKeyMapLock sync.RWMutex
	// Store a reference leaseId -> window key
	leaseIdToKeyMap map[string]string

	syncCircuitBreaker     circuitbreaker.CircuitBreaker[*connect.Response[ratelimitv1.PushPullResponse]]
	mitigateCircuitBreaker circuitbreaker.CircuitBreaker[*connect.Response[ratelimitv1.MitigateResponse]]
}

type Config struct {
	Logger  logging.Logger
	Metrics metrics.Metrics
	Cluster cluster.Cluster
}

func New(cfg Config) (*service, error) {

	s := &service{
		logger:             cfg.Logger,
		cluster:            cfg.Cluster,
		metrics:            cfg.Metrics,
		consistencyChecker: newConsistencyChecker(cfg.Logger),
		peersMu:            sync.RWMutex{},
		peers:              map[string]ratelimitv1connect.RatelimitServiceClient{},
		// Only set if we have a cluster
		syncBuffer:          nil,
		mitigateBuffer:      nil,
		shutdownCh:          make(chan struct{}),
		bucketsMu:           sync.RWMutex{},
		buckets:             make(map[string]*bucket),
		leaseIdToKeyMapLock: sync.RWMutex{},
		leaseIdToKeyMap:     make(map[string]string),

		mitigateCircuitBreaker: circuitbreaker.New[*connect.Response[ratelimitv1.MitigateResponse]](
			"ratelimit.broadcastMitigation",
			circuitbreaker.WithLogger(cfg.Logger),
			circuitbreaker.WithCyclicPeriod(10*time.Second),
			circuitbreaker.WithTimeout(time.Minute),
			circuitbreaker.WithMaxRequests(100),
			circuitbreaker.WithTripThreshold(50),
		),
		syncCircuitBreaker: circuitbreaker.New[*connect.Response[ratelimitv1.PushPullResponse]](
			"ratelimit.syncWithOrigin",
			circuitbreaker.WithLogger(cfg.Logger),
			circuitbreaker.WithCyclicPeriod(10*time.Second),
			circuitbreaker.WithTimeout(time.Minute),
			circuitbreaker.WithMaxRequests(100),
			circuitbreaker.WithTripThreshold(50),
		),
	}

	repeat.Every(time.Minute, s.removeExpiredIdentifiers)

	if cfg.Cluster != nil {
		s.mitigateBuffer = make(chan mitigateWindowRequest, 100000)
		s.syncBuffer = make(chan syncWithOriginRequest, 100000)
		// Process the individual requests to the origin and update local state
		// We're using 128 goroutines to parallelise the network requests'
		s.logger.Info().Msg("starting background jobs")
		for range 128 {
			go func() {
				for {
					select {
					case <-s.shutdownCh:
						return
					case req := <-s.syncBuffer:
						s.syncWithOrigin(req)
					case req := <-s.mitigateBuffer:
						s.broadcastMitigation(req)
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
