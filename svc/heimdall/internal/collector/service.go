package collector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/repeat"
	"github.com/unkeyed/unkey/svc/heimdall/pkg/metrics"
	corelisters "k8s.io/client-go/listers/core/v1"
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

// Config holds the configuration for the kubelet collector.
type Config struct {
	CH        clickhouse.Bufferer
	PodLister corelisters.PodLister
	NodeIP    string
	Region    string
	Platform  string
}

// Collector scrapes kubelet stats and writes resource usage samples to ClickHouse.
type Collector struct {
	ch              clickhouse.Bufferer
	podLister       corelisters.PodLister
	prevReadings    map[string]cpuReading
	kubeletStatsURL string
	region          string
	platform        string

	// mu guards collect() to prevent CollectOnce and regular ticks from overlapping.
	// If they overlap, the CPU delta computation uses a tiny time interval with a tiny
	// counter delta, producing an inflated rate.
	mu sync.Mutex
}

// New creates a new kubelet collector.
func New(cfg Config) *Collector {
	return &Collector{
		ch:              cfg.CH,
		podLister:       cfg.PodLister,
		prevReadings:    make(map[string]cpuReading),
		kubeletStatsURL: fmt.Sprintf("https://%s:10250/stats/summary", cfg.NodeIP),
		region:          cfg.Region,
		platform:        cfg.Platform,
	}
}

// Run starts the collection loop at the given interval.
func (c *Collector) Run(ctx context.Context, interval time.Duration) error {
	stop := repeat.Every(interval, func() {
		c.collectWithMetrics(ctx)
	})

	<-ctx.Done()
	stop()
	return ctx.Err()
}

// CollectPod triggers an immediate kubelet fetch and processes a single pod.
// Called by the lifecycle tracker when a pod starts or enters Terminating
// to grab a counter reading as close to the lifecycle event as possible.
// Only the named pod is processed — all other pods in the kubelet response are skipped.
func (c *Collector) CollectPod(ctx context.Context, podName string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	summary, err := c.fetchSummary(ctx)
	if err != nil {
		logger.Error("immediate collection failed", "pod", podName, "error", err.Error())
		return
	}

	c.processPod(summary, podName)
}

func (c *Collector) collectWithMetrics(ctx context.Context) {
	start := time.Now()

	// Prevent overlap between CollectOnce and regular ticks.
	// TryLock: if a collection is already running, skip this one.
	if !c.mu.TryLock() {
		logger.Info("skipping collection, previous tick still running")
		return
	}
	defer c.mu.Unlock()

	err := c.collect(ctx)

	duration := time.Since(start).Seconds()
	metrics.CollectionDuration.Observe(duration)

	if err != nil {
		metrics.CollectionTotal.WithLabelValues("error").Inc()
		logger.Error("kubelet collection failed", "error", err.Error())
	} else {
		metrics.CollectionTotal.WithLabelValues("success").Inc()
	}
}
