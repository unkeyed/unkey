package logdrain

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"connectrpc.com/connect"
	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/buildinfo/metrics"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/otel"
	"github.com/unkeyed/unkey/pkg/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
	"github.com/unkeyed/unkey/pkg/rpc/interceptor"
	"github.com/unkeyed/unkey/pkg/runner"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/logdrain/internal/db"
	"github.com/unkeyed/unkey/svc/logdrain/internal/engine"
	"github.com/unkeyed/unkey/svc/logdrain/internal/lease"
	"github.com/unkeyed/unkey/svc/logdrain/internal/source"
)

// Run starts the logdrain service and blocks until ctx is cancelled or startup fails.
func Run(ctx context.Context, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("bad config: %w", err)
	}
	if cfg.InstanceID == "" {
		cfg.InstanceID = uid.New(uid.InstancePrefix)
	}
	leaseID := uid.New("")
	reg := promclient.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	//nolint:exhaustruct
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	reg.MustRegister(prometheus.NewSystemMetricsCollector())
	lazy.SetRegistry(reg)
	buildinfometrics.Register("logdrain")

	var shutdown func(context.Context) error
	var err error
	if cfg.Observability.Tracing != nil {
		// PrometheusGatherer stays nil like ctrl: metrics are scraped from
		// /metrics, only logs and traces ship over OTLP.
		shutdown, err = otel.InitGrafana(ctx, otel.Config{
			Application:        "logdrain",
			InstanceID:         cfg.InstanceID,
			CloudRegion:        cfg.Region,
			TraceSampleRate:    cfg.Observability.Tracing.SampleRate,
			PrometheusGatherer: nil,
		})
		if err != nil {
			return fmt.Errorf("initialize grafana: %w", err)
		}
	}
	r := runner.New()
	defer r.Recover()
	r.DeferCtx(shutdown)
	// Single HTTP server on the metrics port serving both Prometheus metrics
	// and the kubelet probe endpoints, like heimdall. Logdrain receives no
	// traffic, so the probes share the metrics port instead of a dedicated
	// listener.
	//
	// Paths:
	//   GET /metrics        Prometheus scrape endpoint (serves reg)
	//   GET /health/live    kubelet liveness probe
	//   GET /health/ready   kubelet readiness probe
	//   GET /health/startup kubelet startup probe
	//
	// Startup (config validation, MySQL, ClickHouse, Vault client) runs
	// synchronously before r.Wait() flips health.started to true, so no
	// extra readiness checks are registered: /ready returning 200 already
	// means the process initialized and is not shutting down. Delivery
	// health stays a metrics concern, not a probe concern; the kubelet must
	// not restart a pod because a customer endpoint is down.
	if cfg.Observability.Metrics != nil && cfg.Observability.Metrics.PrometheusPort > 0 {
		mux := http.NewServeMux()
		//nolint:exhaustruct
		mux.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
		r.RegisterHealth(mux, "/health")

		port := cfg.Observability.Metrics.PrometheusPort
		listener, listenErr := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if listenErr != nil {
			return fmt.Errorf("listen on metrics port %d: %w", port, listenErr)
		}
		server := &http.Server{
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		}
		r.DeferCtx(server.Shutdown)
		r.Go(func(_ context.Context) error {
			logger.Info("metrics+health server started", "port", port)
			serveErr := server.Serve(listener)
			if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
				return fmt.Errorf("metrics+health server failed: %w", serveErr)
			}
			return nil
		})
	}
	database, err := db.New(cfg.Database, sqlcomment.ForService("logdrain", cfg.Region))
	if err != nil {
		return fmt.Errorf("create database: %w", err)
	}
	r.Defer(database.Close)
	ch, err := clickhouse.New(clickhouse.Config{URL: cfg.ClickHouse.URL})
	if err != nil {
		return fmt.Errorf("create clickhouse client: %w", err)
	}
	r.Defer(ch.Close)
	deliveries := clickhouse.NewBuffer[schema.LogdrainDeliveryV1](ch, clickhouse.BufferConfig{
		Name:          "logdrain_deliveries",
		BatchSize:     1_000,
		BufferSize:    2_000,
		FlushInterval: 2 * time.Second,
		Consumers:     1,
		Drop:          true,
		OnFlushError:  nil,
	})
	r.Defer(func() error { deliveries.Close(); return nil })
	vaultClient := vault.NewConnectVaultServiceClient(vaultv1connect.NewVaultServiceClient(
		http.DefaultClient,
		cfg.Vault.URL,
		connect.WithInterceptors(interceptor.NewHeaderInjector(map[string]string{
			"Authorization": "Bearer " + cfg.Vault.Token,
		})),
	))
	serviceClock := clock.New()
	leasing, err := lease.New(lease.Config{
		DB:      database,
		LeaseID: leaseID,
		Clock:   serviceClock,
	})
	if err != nil {
		return fmt.Errorf("build lease service: %w", err)
	}
	delivery, err := engine.New(engine.Config{
		DB:                          database,
		LeaseID:                     leaseID,
		Source:                      source.NewAuditLogs(ch),
		Vault:                       vaultClient,
		Deliveries:                  deliveries,
		Clock:                       serviceClock,
		PollInterval:                cfg.PollInterval,
		WatermarkLag:                cfg.WatermarkLag,
		BatchSize:                   cfg.BatchSize,
		PauseThreshold:              cfg.PauseThreshold,
		MaxConcurrentDrains:         cfg.MaxConcurrentDrains,
		WorkQueueSize:               0,
		UnsafeAllowPrivateEndpoints: cfg.InsecureAllowPrivateEndpoints,
	})
	if err != nil {
		return fmt.Errorf("build delivery engine: %w", err)
	}
	r.Go(leasing.Run)
	r.Go(delivery.Run)
	logger.Info("logdrain service started", "node_id", cfg.InstanceID, "lease_id", leaseID)
	if err := r.Wait(ctx); err != nil {
		return fmt.Errorf("logdrain shutdown: %w", err)
	}
	return nil
}
