package vault

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
	"github.com/unkeyed/unkey/pkg/buildinfo"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/otel"
	"github.com/unkeyed/unkey/pkg/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
	"github.com/unkeyed/unkey/pkg/runner"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/vault/internal/storage"
	storagemiddleware "github.com/unkeyed/unkey/svc/vault/internal/storage/middleware"
	"github.com/unkeyed/unkey/svc/vault/internal/vault"
)

func Run(ctx context.Context, cfg Config) error {
	err := cfg.Validate()
	if err != nil {
		return fmt.Errorf("bad config: %w", err)
	}
	if cfg.InstanceID == "" {
		cfg.InstanceID = uid.New(uid.InstancePrefix)
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
			Application:        "vault",
			InstanceID:         cfg.InstanceID,
			CloudRegion:        cfg.Region,
			TraceSampleRate:    cfg.Observability.Tracing.SampleRate,
			PrometheusGatherer: nil,
		})
		if err != nil {
			return fmt.Errorf("unable to init grafana: %w", err)
		}
	}

	r := runner.New()
	defer r.Recover()

	r.DeferCtx(shutdownGrafana)

	reg := promclient.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	//nolint:exhaustruct
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	reg.MustRegister(prometheus.NewSystemMetricsCollector())
	lazy.SetRegistry(reg)
	buildinfo.RegisterBuildInfoMetrics("vault")

	if cfg.Observability.Metrics != nil && cfg.Observability.Metrics.PrometheusPort > 0 {
		prom, promErr := prometheus.NewWithRegistry(reg)
		if promErr != nil {
			return fmt.Errorf("unable to start prometheus: %w", promErr)
		}

		promListener, listenErr := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Observability.Metrics.PrometheusPort))
		if listenErr != nil {
			return fmt.Errorf("unable to listen on port %d: %w", cfg.Observability.Metrics.PrometheusPort, listenErr)
		}

		r.DeferCtx(prom.Shutdown)
		r.Go(func(ctx context.Context) error {
			serveErr := prom.Serve(ctx, promListener)
			if serveErr != nil && !errors.Is(serveErr, context.Canceled) {
				return fmt.Errorf("prometheus server failed: %w", serveErr)
			}
			return nil
		})
	}

	// Create the connect handler
	mux := http.NewServeMux()
	r.RegisterHealth(mux)

	store, storeName, err := newStorage(cfg.Storage)
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}

	store = storagemiddleware.WithTracing(storeName, store)
	v, err := vault.New(vault.Config{
		Storage:           store,
		MasterKey:         cfg.Encryption.MasterKey,
		PreviousMasterKey: cfg.Encryption.PreviousMasterKey,
		BearerToken:       cfg.BearerToken,
	})
	if err != nil {
		return fmt.Errorf("unable to create vault service: %w", err)
	}

	mux.Handle(vaultv1connect.NewVaultServiceHandler(v))

	addr := fmt.Sprintf(":%d", cfg.HttpPort)
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
		logger.Info("Starting vault server", "addr", addr)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server failed: %w", err)
		}
		return nil
	})

	if err := r.Wait(ctx); err != nil {
		logger.Error("Shutdown failed", "error", err)
		return err
	}

	return nil
}

// newStorage constructs the storage backend selected by cfg. Validation
// guarantees exactly one of S3 or Disk is set.
func newStorage(cfg StorageConfig) (storage.Storage, string, error) {
	switch {
	case cfg.S3 != nil:
		s, err := storage.NewS3(storage.S3Config{
			S3URL:             cfg.S3.URL,
			S3Bucket:          cfg.S3.Bucket,
			S3AccessKeyID:     cfg.S3.AccessKeyID,
			S3AccessKeySecret: cfg.S3.AccessKeySecret,
		})
		if err != nil {
			return nil, "", fmt.Errorf("s3: %w", err)
		}
		return s, "s3", nil
	case cfg.Disk != nil:
		s, err := storage.NewDisk(cfg.Disk.Path)
		if err != nil {
			return nil, "", fmt.Errorf("disk: %w", err)
		}
		return s, "disk", nil
	default:
		return nil, "", fmt.Errorf("storage: no backend configured")
	}
}
