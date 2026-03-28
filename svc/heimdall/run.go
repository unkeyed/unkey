package heimdall

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/pkg/clickhouse"
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

	collectionInterval, err := time.ParseDuration(cfg.CollectionInterval)
	if err != nil {
		return fmt.Errorf("invalid collection_interval %q: %w", cfg.CollectionInterval, err)
	}

	logger.Info("starting heimdall",
		"region", cfg.Region,
		"platform", cfg.Platform,
		"collection_interval", collectionInterval.String(),
	)

	ch, err := clickhouse.New(clickhouse.Config{URL: cfg.ClickHouseURL})
	if err != nil {
		return fmt.Errorf("unable to create clickhouse client: %w", err)
	}

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
		return ch.Close()
	})

	factory.Start(ctx.Done())
	factory.WaitForCacheSync(ctx.Done())

	kc := collector.New(collector.Config{
		CH:            ch,
		PodLister:     podLister,
		MetricsClient: metricsClient,
		NodeName:      cfg.NodeName,
		Region:        cfg.Region,
		Platform:      cfg.Platform,
	})

	r.Go(func(ctx context.Context) error {
		return kc.Run(ctx, collectionInterval)
	})

	return r.Wait(ctx)
}
