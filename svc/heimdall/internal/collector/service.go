package collector

import (
	"context"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/repeat"
	"github.com/unkeyed/unkey/svc/heimdall/pkg/metrics"
	corelisters "k8s.io/client-go/listers/core/v1"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

const (
	LabelManagedBy  = "app.kubernetes.io/managed-by"
	LabelComponent  = "app.kubernetes.io/component"
	LabelWorkspace  = "unkey.com/workspace.id"
	LabelProject    = "unkey.com/project.id"
	LabelApp        = "unkey.com/app.id"
	LabelEnv        = "unkey.com/environment.id"
	LabelDeployment = "unkey.com/deployment.id"
)

type Config struct {
	CH            clickhouse.Bufferer
	PodLister     corelisters.PodLister
	MetricsClient metricsv.Interface
	Region        string
	Platform      string
}

// Collector writes resource snapshots to ClickHouse every collection interval.
type Collector struct {
	ch            clickhouse.Bufferer
	podLister     corelisters.PodLister
	metricsClient metricsv.Interface
	region        string
	platform      string
	mu            sync.Mutex
}

func New(cfg Config) *Collector {
	return &Collector{
		ch:            cfg.CH,
		podLister:     cfg.PodLister,
		metricsClient: cfg.MetricsClient,
		region:        cfg.Region,
		platform:      cfg.Platform,
	}
}

func (c *Collector) Run(ctx context.Context, interval time.Duration) error {
	stop := repeat.Every(interval, func() {
		c.collectWithMetrics(ctx)
	})

	<-ctx.Done()
	stop()
	return ctx.Err()
}

func (c *Collector) collectWithMetrics(ctx context.Context) {
	start := time.Now()

	if !c.mu.TryLock() {
		logger.Info("skipping collection, previous tick still running")
		return
	}
	defer c.mu.Unlock()

	err := c.collect(ctx)

	metrics.CollectionDuration.Observe(time.Since(start).Seconds())

	if err != nil {
		metrics.CollectionTotal.WithLabelValues("error").Inc()
		logger.Error("collection failed", "error", err.Error())
	} else {
		metrics.CollectionTotal.WithLabelValues("success").Inc()
	}
}
