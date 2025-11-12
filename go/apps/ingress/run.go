package ingress

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"runtime/debug"

	"github.com/unkeyed/unkey/go/apps/ingress/routes"
	"github.com/unkeyed/unkey/go/apps/ingress/services/caches"
	"github.com/unkeyed/unkey/go/apps/ingress/services/certmanager"
	"github.com/unkeyed/unkey/go/apps/ingress/services/deployments"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/prometheus"
	"github.com/unkeyed/unkey/go/pkg/shutdown"
	pkgtls "github.com/unkeyed/unkey/go/pkg/tls"
	"github.com/unkeyed/unkey/go/pkg/vault"
	"github.com/unkeyed/unkey/go/pkg/vault/storage"
	"github.com/unkeyed/unkey/go/pkg/version"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

// Run starts the Ingress server
// Ingress is our multi-tenant ingress service that:
// - Accepts requests from NLBs (<region>.aws.unkey.app)
// - Terminates TLS if the deployment exists in its region
// - Forwards requests to the per-tenant/environment gateway service
// - OR forwards to another region's NLB if no local deployment exists
func Run(ctx context.Context, cfg Config) error {
	err := cfg.Validate()
	if err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	logger := logging.New()
	if cfg.IngressID != "" {
		logger = logger.With(slog.String("instanceID", cfg.IngressID))
	}

	if cfg.Platform != "" {
		logger = logger.With(slog.String("platform", cfg.Platform))
	}

	if cfg.Region != "" {
		logger = logger.With(slog.String("region", cfg.Region))
	}

	if version.Version != "" {
		logger = logger.With(slog.String("version", version.Version))
	}

	// Catch any panics now after we have a logger but before we start the server
	defer func() {
		if r := recover(); r != nil {
			logger.Error("panic",
				"panic", r,
				"stack", string(debug.Stack()),
			)
		}
	}()

	shutdowns := shutdown.New()
	clk := clock.New()
	_ = clk

	if cfg.OtelEnabled {
		grafanaErr := otel.InitGrafana(
			ctx,
			otel.Config{
				Application:     "gate",
				Version:         version.Version,
				InstanceID:      cfg.IngressID,
				CloudRegion:     cfg.Region,
				TraceSampleRate: cfg.OtelTraceSamplingRate,
			},
			shutdowns,
		)

		if grafanaErr != nil {
			return fmt.Errorf("unable to init grafana: %w", grafanaErr)
		}
	}

	if cfg.PrometheusPort > 0 {
		prom, promErr := prometheus.New(prometheus.Config{
			Logger: logger,
		})
		if promErr != nil {
			return fmt.Errorf("unable to start prometheus: %w", promErr)
		}

		go func() {
			var promListener net.Listener
			promListener, err = net.Listen("tcp", fmt.Sprintf(":%d", cfg.PrometheusPort))
			if err != nil {
				panic(err)
			}
			promListenErr := prom.Serve(ctx, promListener)
			if promListenErr != nil {
				panic(promListenErr)
			}
		}()
	}

	var vaultSvc *vault.Service
	if len(cfg.VaultMasterKeys) > 0 && cfg.VaultS3 != nil {
		var vaultStorage storage.Storage
		vaultStorage, err = storage.NewS3(storage.S3Config{
			Logger:            logger,
			S3URL:             cfg.VaultS3.S3URL,
			S3Bucket:          cfg.VaultS3.S3Bucket,
			S3AccessKeyID:     cfg.VaultS3.S3AccessKeyID,
			S3AccessKeySecret: cfg.VaultS3.S3AccessKeySecret,
		})
		if err != nil {
			return fmt.Errorf("unable to create vault storage: %w", err)
		}

		vaultSvc, err = vault.New(vault.Config{
			Logger:     logger,
			Storage:    vaultStorage,
			MasterKeys: cfg.VaultMasterKeys,
		})
		if err != nil {
			return fmt.Errorf("unable to create vault service: %w", err)
		}
	}

	partitionedDB, err := db.New(db.Config{
		PrimaryDSN:  cfg.DatabasePrimary,
		ReadOnlyDSN: cfg.DatabaseReadonlyReplica,
		Logger:      logger,
	})
	if err != nil {
		return fmt.Errorf("unable to create partitioned db: %w", err)
	}
	shutdowns.Register(partitionedDB.Close)

	// Initialize caches
	cachesInstance, err := caches.New(caches.Config{
		Logger: logger,
		Clock:  clk,
	})
	if err != nil {
		return fmt.Errorf("unable to create caches: %w", err)
	}

	// Initialize certificate manager for dynamic TLS
	var certManager certmanager.Service
	if vaultSvc != nil {
		certManager = certmanager.New(certmanager.Config{
			Logger:              logger,
			DB:                  partitionedDB,
			TLSCertificateCache: cachesInstance.TLSCertificate,
			DefaultCertDomain:   cfg.BaseDomain,
			Vault:               vaultSvc,
		})
	}

	// Initialize deployment lookup service
	deploymentSvc, err := deployments.New(deployments.Config{
		Logger: logger,
	})
	if err != nil {
		return fmt.Errorf("unable to create deployment service: %w", err)
	}

	// Create TLS config with dynamic certificate loading
	var tlsConfig *pkgtls.Config
	if cfg.EnableTLS && certManager != nil {
		//nolint:exhaustruct
		tlsConfig = &tls.Config{
			GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return certManager.GetCertificate(context.Background(), hello.ServerName)
			},
			MinVersion: tls.VersionTLS13,
		}
	}

	// Create zen server
	srv, err := zen.New(zen.Config{
		Logger: logger,
		TLS:    tlsConfig,
	})
	if err != nil {
		return fmt.Errorf("unable to create server: %w", err)
	}
	shutdowns.RegisterCtx(srv.Shutdown)

	// Register all routes
	routes.Register(srv, &routes.Services{
		Logger:            logger,
		DeploymentService: deploymentSvc,
		CurrentRegion:     cfg.Region,
		BaseDomain:        cfg.BaseDomain,
	})

	// Start HTTPS server
	if cfg.HttpsPort > 0 {
		httpsListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.HttpsPort))
		if err != nil {
			return fmt.Errorf("unable to create HTTPS listener: %w", err)
		}

		go func() {
			logger.Info("HTTPS ingress server started", "addr", httpsListener.Addr().String())
			serveErr := srv.Serve(ctx, httpsListener)
			if serveErr != nil {
				logger.Error("HTTPS server error", "error", serveErr)
			}
		}()
	}

	// Start HTTP server (for ACME challenges or testing)
	if cfg.HttpPort > 0 {
		httpListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.HttpPort))
		if err != nil {
			return fmt.Errorf("unable to create HTTP listener: %w", err)
		}

		go func() {
			logger.Info("HTTP ingress server started", "addr", httpListener.Addr().String())
			serveErr := srv.Serve(ctx, httpListener)
			if serveErr != nil {
				logger.Error("HTTP server error", "error", serveErr)
			}
		}()
	}

	logger.Info("Ingress server initialized", "region", cfg.Region, "baseDomain", cfg.BaseDomain)

	// Wait for either OS signals or context cancellation, then shutdown
	if err := shutdowns.WaitForSignal(ctx, 0); err != nil {
		return fmt.Errorf("shutdown failed: %w", err)
	}

	logger.Info("Ingress server shut down successfully")
	return nil
}
