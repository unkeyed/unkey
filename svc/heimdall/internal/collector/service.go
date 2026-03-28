package collector

import (
	"context"
	"net"
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
	NodeName      string
	Region        string
	Platform      string
}

// Collector writes resource snapshots to ClickHouse every collection interval.
// It runs as a DaemonSet — one per node. Each instance collects:
//   - CPU/memory from Metrics Server (filtered to local node's pods)
//   - Network egress from local conntrack table (per-connection bytes)
//   - Pod labels + limits from informer cache
type Collector struct {
	ch             clickhouse.Bufferer
	podLister      corelisters.PodLister
	metricsClient  metricsv.Interface
	nodeName       string
	region         string
	platform       string
	internalCIDRs  []*net.IPNet
	prevEgress     map[string]podEgress // track previous conntrack totals for delta
	mu             sync.Mutex
}

func New(cfg Config) *Collector {
	return &Collector{
		ch:            cfg.CH,
		podLister:     cfg.PodLister,
		metricsClient: cfg.MetricsClient,
		nodeName:      cfg.NodeName,
		region:        cfg.Region,
		platform:      cfg.Platform,
		internalCIDRs: defaultInternalCIDRs(),
		prevEgress:    make(map[string]podEgress),
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
