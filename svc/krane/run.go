package krane

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"connectrpc.com/connect"
	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/otel"
	"github.com/unkeyed/unkey/pkg/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
	"github.com/unkeyed/unkey/pkg/repeat"
	"github.com/unkeyed/unkey/pkg/rpc/interceptor"
	"github.com/unkeyed/unkey/pkg/runner"
	pkgversion "github.com/unkeyed/unkey/pkg/version"
	"github.com/unkeyed/unkey/svc/krane/internal/cilium"
	"github.com/unkeyed/unkey/svc/krane/internal/deployment"
	"github.com/unkeyed/unkey/svc/krane/internal/sentinel"
	"github.com/unkeyed/unkey/svc/krane/internal/watcher"
	"github.com/unkeyed/unkey/svc/krane/pkg/controlplane"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Run starts the krane agent server with the provided configuration.
//
// It initializes Kubernetes clients, vault for secrets decryption, controller
// loops (cilium, deployment, sentinel), an HTTP health endpoint, and optional
// Prometheus metrics. It blocks until the context is cancelled or a fatal error
// occurs.
func Run(ctx context.Context, cfg Config) error {
	err := cfg.Validate()
	if err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	if cfg.Observability.Logging != nil {
		logger.SetSampler(logger.TailSampler{
			SlowThreshold: cfg.Observability.Logging.SlowThreshold,
			SampleRate:    cfg.Observability.Logging.SampleRate,
		})
	}

	var shutdownGrafana func(context.Context) error
	if cfg.Observability.Tracing != nil {
		shutdownGrafana, err = otel.InitGrafana(ctx, otel.Config{
			Application:     "krane",
			Version:         pkgversion.Version,
			InstanceID:      cfg.InstanceID,
			CloudRegion:     cfg.Region,
			TraceSampleRate: cfg.Observability.Tracing.SampleRate,
		})
		if err != nil {
			return fmt.Errorf("unable to init grafana: %w", err)
		}
	}

	// Add base attributes to global logger
	logger.AddBaseAttrs(slog.GroupAttrs("instance",
		slog.String("id", cfg.InstanceID),
		slog.String("region", cfg.Region),
		slog.String("version", pkgversion.Version),
	))

	r := runner.New()
	defer r.Recover()

	r.DeferCtx(shutdownGrafana)

	reg := promclient.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	//nolint:exhaustruct
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	lazy.SetRegistry(reg)

	cluster := controlplane.NewClient(controlplane.ClientConfig{
		URL:         cfg.Control.URL,
		BearerToken: cfg.Control.Token,
		Region:      cfg.Region,
		Platform:    cfg.Platform,
	})

	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to create in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(inClusterConfig)
	if err != nil {
		return fmt.Errorf("failed to create k8s clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(inClusterConfig)
	if err != nil {
		return fmt.Errorf("failed to create k8s dynamic client: %w", err)
	}

	// Create vault client for deploy-time secret decryption
	var vaultClient vault.VaultServiceClient
	if cfg.Vault.URL != "" {
		vaultClient = vault.NewConnectVaultServiceClient(vaultv1connect.NewVaultServiceClient(
			http.DefaultClient,
			cfg.Vault.URL,
			connect.WithInterceptors(interceptor.NewHeaderInjector(map[string]string{
				"Authorization": "Bearer " + cfg.Vault.Token,
			})),
		))
		logger.Info("Vault client initialized", "url", cfg.Vault.URL)
	}

	// Start the cilium controller (independent control loop)
	ciliumCtrl := cilium.New(cilium.Config{
		ClientSet:     clientset,
		DynamicClient: dynamicClient,
		Cluster:       cluster,
		Region:        cfg.Region,
	})
	if err := ciliumCtrl.Start(ctx); err != nil {
		return fmt.Errorf("failed to start cilium controller: %w", err)
	}
	r.Defer(ciliumCtrl.Stop)

	// Build registry config for pull secret creation
	var registryCfg *deployment.RegistryConfig
	if cfg.Registry != nil {
		registryCfg = deployment.NewRegistryConfig(cfg.Registry.URL, cfg.Registry.Username, cfg.Registry.Password)
	}

	// Cache for deduplicating deployment status reports. Entries auto-expire
	// so deleted ReplicaSets don't leak memory.
	fingerprintCache, err := cache.New(cache.Config[string, string]{
		Fresh:    5 * time.Minute,
		Stale:    10 * time.Minute,
		MaxSize:  10_000,
		Resource: "deployment_fingerprints",
		Clock:    clock.New(),
	})
	if err != nil {
		return fmt.Errorf("failed to create fingerprint cache: %w", err)
	}

	// Start the deployment controller (independent control loop)
	deploymentCtrl := deployment.New(deployment.Config{
		ClientSet:        clientset,
		DynamicClient:    dynamicClient,
		Cluster:          cluster,
		Region:           cfg.Region,
		Platform:         cfg.Platform,
		Vault:            vaultClient,
		Registry:         registryCfg,
		Fingerprints:     fingerprintCache,
		StorageClassName: cfg.StorageClassName,
	})
	if err := deploymentCtrl.Start(ctx); err != nil {
		return fmt.Errorf("failed to start deployment controller: %w", err)
	}
	r.Defer(deploymentCtrl.Stop)

	// Start the sentinel controller (independent control loop)
	sentinelCtrl := sentinel.New(sentinel.Config{
		ClientSet:     clientset,
		DynamicClient: dynamicClient,
		Cluster:       cluster,
		Region:        cfg.Region,
		Platform:      cfg.Platform,
	})
	if err := sentinelCtrl.Start(ctx); err != nil {
		return fmt.Errorf("failed to start sentinel controller: %w", err)
	}
	r.Defer(sentinelCtrl.Stop)

	// Start the unified syncer that consumes WatchDeploymentChanges and
	// dispatches events to the deployment, sentinel, and cilium controllers.
	w := watcher.New(watcher.Config{
		Cluster:     cluster,
		Deployments: deploymentCtrl,
		Sentinels:   sentinelCtrl,
		Cilium:      ciliumCtrl,
		Region:      cfg.Region,
		Platform:    cfg.Platform,
	})
	r.Go(w.Watch)

	// Start heartbeat loop to register this cluster with the control plane
	stopHeartbeat := repeat.Every(30*time.Second, func() {
		if _, err := cluster.Heartbeat(ctx, &ctrlv1.HeartbeatRequest{
			Region:   cfg.Region,
			Platform: cfg.Platform,
		}); err != nil {
			logger.Warn("heartbeat failed", "error", err)
		}
	})
	r.Defer(func() error { stopHeartbeat(); return nil })

	// Create the connect handler
	mux := http.NewServeMux()
	r.RegisterHealth(mux)

	addr := fmt.Sprintf(":%d", cfg.RPCPort)
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
		// Do not set timeouts here, our streaming rpcs will get canceled too frequently
	}

	// Register server shutdown
	r.DeferCtx(server.Shutdown)

	// Start server
	r.Go(func(ctx context.Context) error {
		logger.Info("Starting control server", "addr", addr, "tls")

		err := server.ListenAndServe()

		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server failed: %w", err)
		}
		return nil
	})

	if cfg.Observability.Metrics != nil && cfg.Observability.Metrics.PrometheusPort > 0 {

		prom, err := prometheus.NewWithRegistry(reg)
		if err != nil {
			return fmt.Errorf("failed to create prometheus server: %w", err)
		}

		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Observability.Metrics.PrometheusPort))
		if err != nil {
			return fmt.Errorf("unable to listen on port %d: %w", cfg.Observability.Metrics.PrometheusPort, err)
		}

		r.DeferCtx(prom.Shutdown)
		r.Go(func(ctx context.Context) error {
			logger.Info("prometheus started", "port", cfg.Observability.Metrics.PrometheusPort)
			if serveErr := prom.Serve(ctx, ln); serveErr != nil && !errors.Is(serveErr, context.Canceled) {
				return fmt.Errorf("failed to start prometheus server: %w", serveErr)
			}
			return nil
		})

	}

	// Wait for signal and handle shutdown
	logger.Info("Krane server started successfully")
	if err := r.Wait(ctx); err != nil {
		logger.Error("Shutdown failed", "error", err)
		return err
	}

	logger.Info("krane server shut down successfully")
	return nil
}
