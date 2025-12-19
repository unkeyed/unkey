package krane

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	deploymentreflector "github.com/unkeyed/unkey/go/apps/krane/deployment_reflector"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/controlplane"
	sentinelreflector "github.com/unkeyed/unkey/go/apps/krane/sentinel_reflector"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/prometheus"
	"github.com/unkeyed/unkey/go/pkg/shutdown"
	pkgversion "github.com/unkeyed/unkey/go/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func Run(ctx context.Context, cfg Config) error {
	err := cfg.Validate()
	if err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	shutdowns := shutdown.New()

	logger := logging.New()
	if cfg.InstanceID != "" {
		logger = logger.With(slog.String("instanceID", cfg.InstanceID))
	}
	if cfg.Region != "" {
		logger = logger.With(slog.String("region", cfg.Region))
	}
	if pkgversion.Version != "" {
		logger = logger.With(slog.String("version", pkgversion.Version))
	}

	cluster := controlplane.NewClient(controlplane.ClientConfig{
		URL:         cfg.ControlPlaneURL,
		BearerToken: cfg.ControlPlaneBearer,
		Region:      cfg.Region,
		Shard:       cfg.Shard,
	})

	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to create in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(inClusterConfig)
	if err != nil {
		return fmt.Errorf("failed to create k8s clientset: %w", err)
	}

	sc, err := sentinelreflector.New(sentinelreflector.Config{
		Logger:     logger,
		ClientSet:  clientset,
		Cluster:    cluster,
		InstanceID: cfg.InstanceID,
		Region:     cfg.Region,
		Shard:      cfg.Shard,
	})
	if err != nil {
		return fmt.Errorf("failed to create sentinel controller: %w", err)
	}
	go sc.Start()
	shutdowns.Register(sc.Stop)

	dc, err := deploymentreflector.New(deploymentreflector.Config{
		Logger:     logger,
		ClientSet:  clientset,
		Cluster:    cluster,
		InstanceID: cfg.InstanceID,
		Region:     cfg.Region,
		Shard:      cfg.Shard,
	})
	if err != nil {
		return fmt.Errorf("failed to create sentinel controller: %w", err)
	}
	go dc.Start()
	shutdowns.Register(dc.Stop)

	if cfg.PrometheusPort > 0 {

		prom, err := prometheus.New(prometheus.Config{
			Logger: logger,
		})
		if err != nil {
			return fmt.Errorf("failed to create prometheus server: %w", err)
		}

		shutdowns.RegisterCtx(prom.Shutdown)
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.PrometheusPort))
		if err != nil {
			return fmt.Errorf("unable to listen on port %d: %w", cfg.PrometheusPort, err)
		}
		go func() {
			logger.Info("prometheus started", "port", cfg.PrometheusPort)
			if err := prom.Serve(ctx, ln); err != nil {
				logger.Error("failed to start prometheus server", "error", err)
			}
		}()

	}
	// Wait for signal and handle shutdown
	logger.Info("Krane server started successfully")
	if err := shutdowns.WaitForSignal(ctx); err != nil {
		logger.Error("Shutdown failed", "error", err)
		return err
	}

	logger.Info("krane server shut down successfully")
	return nil
}
