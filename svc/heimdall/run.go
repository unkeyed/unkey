package heimdall

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/runner"
	"github.com/unkeyed/unkey/svc/heimdall/internal/collector"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

func Run(ctx context.Context, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	logger.Info("starting heimdall",
		"region", cfg.Region,
		"platform", cfg.Platform,
		"collection_interval", cfg.CollectionInterval.String(),
	)

	ch, err := clickhouse.New(clickhouse.Config{URL: cfg.ClickHouseURL})
	if err != nil {
		return fmt.Errorf("unable to create clickhouse client: %w", err)
	}

	snapshotBuffer := clickhouse.NewBuffer[schema.ResourceSnapshot](ch, "default.instance_resource_snapshots_v1", clickhouse.BufferConfig{
		Name:          "resource_snapshots",
		Drop:          false,
		BatchSize:     10_000,
		BufferSize:    50_000,
		FlushInterval: 5 * time.Second,
		Consumers:     1,
		OnFlushError:  nil,
	})

	k8sCfg, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("getting in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(k8sCfg)
	if err != nil {
		return fmt.Errorf("creating kubernetes client: %w", err)
	}

	metricsClient, err := metricsv.NewForConfig(k8sCfg)
	if err != nil {
		return fmt.Errorf("creating metrics client: %w", err)
	}

	factory := informers.NewSharedInformerFactory(clientset, 0)
	podLister := factory.Core().V1().Pods().Lister()

	r := runner.New()

	r.Defer(func() error {
		snapshotBuffer.Close()
		return ch.Close()
	})

	factory.Start(ctx.Done())
	factory.WaitForCacheSync(ctx.Done())

	kc := collector.New(collector.Config{
		CH:            snapshotBuffer,
		PodLister:     podLister,
		MetricsClient: metricsClient,
		NodeName:      cfg.NodeName,
		Region:        cfg.Region,
		Platform:      cfg.Platform,
	})

	r.Go(func(ctx context.Context) error {
		return kc.Run(ctx, cfg.CollectionInterval)
	})

	return r.Wait(ctx)
}
