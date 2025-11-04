package gw

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/unkeyed/unkey/go/apps/gw/router"
	"github.com/unkeyed/unkey/go/apps/gw/server"
	"github.com/unkeyed/unkey/go/apps/gw/services/caches"
	"github.com/unkeyed/unkey/go/apps/gw/services/certmanager"
	"github.com/unkeyed/unkey/go/apps/gw/services/routing"
	"github.com/unkeyed/unkey/go/apps/gw/services/validation"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/ratelimit"
	"github.com/unkeyed/unkey/go/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/counter"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/prometheus"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/shutdown"
	"github.com/unkeyed/unkey/go/pkg/vault"
	"github.com/unkeyed/unkey/go/pkg/vault/storage"
	"github.com/unkeyed/unkey/go/pkg/version"
)

// nolint:gocognit
func Run(ctx context.Context, cfg Config) error {
	err := cfg.Validate()
	if err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	logger := logging.New()
	if cfg.GatewayID != "" {
		logger = logger.With(slog.String("gatewayID", cfg.GatewayID))
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

	if cfg.OtelEnabled {
		grafanaErr := otel.InitGrafana(
			ctx,
			otel.Config{
				Application:     "gateway",
				Version:         version.Version,
				InstanceID:      cfg.GatewayID,
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

	// Generate local certificate if requested
	if cfg.RequireLocalCert {
		localCertCfg := LocalCertConfig{
			Logger:        logger,
			PartitionedDB: partitionedDB,
			VaultService:  vaultSvc,
			Hostname:      "*.unkey.local",
			WorkspaceID:   "unkey",
		}

		err = generateLocalCertificate(ctx, localCertCfg)
		if err != nil {
			return fmt.Errorf("failed to generate local certificate: %w", err)
		}
	}

	// Create separate non-partitioned database connection for keys service
	var mainDB db.Database
	mainDB, err = db.New(db.Config{
		PrimaryDSN:  cfg.MainDatabasePrimary,
		ReadOnlyDSN: cfg.MainDatabaseReadonlyReplica,
		Logger:      logger,
	})
	if err != nil {
		return fmt.Errorf("unable to create main db: %w", err)
	}
	shutdowns.Register(mainDB.Close)

	var ch clickhouse.ClickHouse = clickhouse.NewNoop()
	if cfg.ClickhouseURL != "" {
		ch, err = clickhouse.New(clickhouse.Config{
			URL:    cfg.ClickhouseURL,
			Logger: logger,
		})
		if err != nil {
			return fmt.Errorf("unable to create clickhouse: %w", err)
		}
	}

	caches, err := caches.New(caches.Config{
		Logger: logger,
		Clock:  clk,
	})
	if err != nil {
		return fmt.Errorf("unable to create caches: %w", err)
	}

	// Use in-memory counter since Redis is nil
	ctr, err := counter.NewRedis(counter.RedisConfig{
		Logger:   logger,
		RedisURL: cfg.RedisURL,
	})
	if err != nil {
		return fmt.Errorf("unable to create counter: %w", err)
	}
	shutdowns.Register(ctr.Close)

	// Create rate limiting service
	rlSvc, err := ratelimit.New(ratelimit.Config{
		Logger:  logger,
		Clock:   clk,
		Counter: ctr,
	})
	if err != nil {
		return fmt.Errorf("unable to create ratelimit service: %w", err)
	}
	shutdowns.Register(rlSvc.Close)

	ulSvc, err := usagelimiter.NewRedisWithCounter(usagelimiter.RedisConfig{
		Logger:  logger,
		DB:      mainDB,
		Counter: ctr,
		TTL:     30 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("unable to create usage limiter service: %w", err)
	}

	// Create key service with non-partitioned database
	keySvc, err := keys.New(keys.Config{
		Logger:       logger,
		DB:           mainDB,
		KeyCache:     caches.VerificationKeyByHash,
		RateLimiter:  rlSvc,
		RBAC:         rbac.New(),
		UsageLimiter: ulSvc,
		Clickhouse:   ch,
		Region:       cfg.Region,
	})
	if err != nil {
		return fmt.Errorf("unable to create key service: %w", err)
	}
	shutdowns.Register(keySvc.Close)

	// Create routing service with partitioned database
	routingService, err := routing.New(routing.Config{
		DB:                 partitionedDB,
		Logger:             logger,
		Clock:              clk,
		GatewayConfigCache: caches.GatewayConfig,
		VMCache:            caches.VM,
	})
	if err != nil {
		return fmt.Errorf("unable to create routing service: %w", err)
	}

	// Create certificate manager with optional default cert domain
	certManager := certmanager.New(certmanager.Config{
		Logger:              logger,
		DB:                  partitionedDB,
		TLSCertificateCache: caches.TLSCertificate,
		DefaultCertDomain:   cfg.DefaultCertDomain,
		Vault:               vaultSvc,
	})

	// Create validation service
	validationService, err := validation.New(validation.Config{
		Logger:           logger,
		OpenAPISpecCache: caches.OpenAPISpec,
	})
	if err != nil {
		return fmt.Errorf("unable to create validation service: %w", err)
	}

	httpClient := &http.Client{}
	acmeClient := ctrlv1connect.NewAcmeServiceClient(httpClient, cfg.CtrlAddr)

	// Create HTTP server for ACME challenges
	challengeSrv, err := server.New(server.Config{
		CertManager: nil,
		Logger:      logger,
		Handler:     nil,
		EnableTLS:   false,
	})
	if err != nil {
		return fmt.Errorf("unable to create challenge server: %w", err)
	}
	shutdowns.RegisterCtx(challengeSrv.Shutdown)

	// Create HTTPS gateway server
	gwSrv, err := server.New(server.Config{
		Logger:      logger,
		Handler:     nil,
		CertManager: certManager,
		EnableTLS:   cfg.EnableTLS,
	})
	if err != nil {
		return fmt.Errorf("unable to create gateway server: %w", err)
	}
	shutdowns.RegisterCtx(gwSrv.Shutdown)

	// Services configuration for both servers
	services := &router.Services{
		Logger:         logger,
		CertManager:    certManager,
		RoutingService: routingService,
		Validation:     validationService,
		ClickHouse:     ch,
		Keys:           keySvc,
		Ratelimit:      nil,
		MainDomain:     cfg.MainDomain,
		AcmeClient:     acmeClient,
		// For now just enable it if we don't do SSL Termination
		HttpProxy: !cfg.EnableTLS,
	}

	// Register routes for HTTP server (ACME challenges)
	err = router.Register(challengeSrv, services, cfg.Region, router.HTTPServer)
	if err != nil {
		return fmt.Errorf("unable to register routes for HTTP server: %w", err)
	}

	// Register routes for HTTPS server (main gateway)
	err = router.Register(gwSrv, services, cfg.Region, router.HTTPSServer)
	if err != nil {
		return fmt.Errorf("unable to register routes for HTTPS server: %w", err)
	}

	if cfg.HttpPort > 0 {
		challengeListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.HttpPort))
		if err != nil {
			return fmt.Errorf("unable to create listener: %w", err)
		}

		go func() {
			logger.Info("HTTP server (ACME challenges) started successfully", "addr", challengeListener.Addr().String())
			serveErr := challengeSrv.Serve(ctx, challengeListener)
			if serveErr != nil {
				panic(serveErr)
			}
		}()
	}

	if cfg.HttpsPort > 0 {
		gwListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.HttpsPort))
		if err != nil {
			return fmt.Errorf("unable to create listener: %w", err)
		}

		go func() {
			logger.Info("HTTPS gateway server started successfully", "addr", gwListener.Addr().String())
			serveErr := gwSrv.Serve(ctx, gwListener)
			if serveErr != nil {
				panic(serveErr)
			}
		}()
	}

	// Wait for either OS signals or context cancellation, then shutdown
	if err := shutdowns.WaitForSignal(ctx, time.Minute); err != nil {
		return fmt.Errorf("shutdown failed: %w", err)
	}

	logger.Info("Gateway server shut down successfully")
	return nil
}
