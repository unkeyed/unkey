package ingress

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/unkeyed/unkey/go/apps/ingress/routes"
	"github.com/unkeyed/unkey/go/apps/ingress/services/caches"
	"github.com/unkeyed/unkey/go/apps/ingress/services/certmanager"
	"github.com/unkeyed/unkey/go/apps/ingress/services/proxy"
	"github.com/unkeyed/unkey/go/apps/ingress/services/router"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/prometheus"
	"github.com/unkeyed/unkey/go/pkg/ptr"
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
// - OR forwards to another region if no local deployment exists
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

	// Create cached clock with millisecond resolution for efficient time tracking
	clk := clock.NewCachedClock(time.Millisecond)
	shutdowns.Register(func() error {
		clk.Close()
		return nil
	})

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

	db, err := db.New(db.Config{
		PrimaryDSN:  cfg.DatabasePrimary,
		ReadOnlyDSN: cfg.DatabaseReadonlyReplica,
		Logger:      logger,
	})
	if err != nil {
		return fmt.Errorf("unable to create partitioned db: %w", err)
	}
	shutdowns.Register(db.Close)

	// Initialize caches
	cache, err := caches.New(caches.Config{
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
			DB:                  db,
			TLSCertificateCache: cache.TLSCertificate,
			Vault:               vaultSvc,
		})
	}

	// Initialize router service
	routerSvc, err := router.New(router.Config{
		Logger:                logger,
		Region:                cfg.Region,
		DB:                    db,
		IngressRouteCache:     cache.IngressRoute,
		GatewaysByEnvironment: cache.GatewaysByEnvironment,
	})
	if err != nil {
		return fmt.Errorf("unable to create router service: %w", err)
	}

	// Initialize proxy service with shared transport for connection pooling
	proxySvc, err := proxy.New(proxy.Config{
		Logger:     logger,
		IngressID:  cfg.IngressID,
		Region:     cfg.Region,
		BaseDomain: cfg.BaseDomain,
		Clock:      clk,
		MaxHops:    cfg.MaxHops,
		// Use defaults for transport settings (200 max idle conns, 90s timeout, etc.)
	})
	if err != nil {
		return fmt.Errorf("unable to create proxy service: %w", err)
	}

	// Create TLS config with dynamic certificate loading
	var tlsConfig *pkgtls.Config
	if cfg.EnableTLS && certManager != nil {
		//nolint:exhaustruct
		tlsConfig = &tls.Config{
			GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return certManager.GetCertificate(context.Background(), hello.ServerName)
			},
			MinVersion: tls.VersionTLS12,
		}
	}

	acmeClient := ctrlv1connect.NewAcmeServiceClient(ptr.P(http.Client{}), cfg.CtrlAddr)
	svcs := &routes.Services{
		Logger:        logger,
		Region:        cfg.Region,
		RouterService: routerSvc,
		ProxyService:  proxySvc,
		Clock:         clk,
		AcmeClient:    acmeClient,
	}

	// Start HTTPS ingress server (main proxy server)
	if cfg.HttpsPort > 0 {
		httpsSrv, httpsErr := zen.New(zen.Config{
			Logger: logger,
			TLS:    tlsConfig,
			// Use longer timeouts for proxy operations
			// WriteTimeout must be longer than the transport's ResponseHeaderTimeout (30s)
			// so that transport timeouts can be caught and handled properly in ErrorHandler
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 60 * time.Second,
		})
		if httpsErr != nil {
			return fmt.Errorf("unable to create HTTPS server: %w", httpsErr)
		}
		shutdowns.RegisterCtx(httpsSrv.Shutdown)

		// Register all ingress routes on HTTPS server
		routes.Register(httpsSrv, svcs)

		httpsListener, httpsListenErr := net.Listen("tcp", fmt.Sprintf(":%d", cfg.HttpsPort))
		if httpsListenErr != nil {
			return fmt.Errorf("unable to create HTTPS listener: %w", httpsListenErr)
		}

		go func() {
			logger.Info("HTTPS ingress server started", "addr", httpsListener.Addr().String())
			serveErr := httpsSrv.Serve(ctx, httpsListener)
			if serveErr != nil {
				logger.Error("HTTPS server error", "error", serveErr)
			}
		}()
	}

	// Start HTTP challenge server (ACME only for Let's Encrypt)
	if cfg.HttpPort > 0 {
		httpSrv, httpErr := zen.New(zen.Config{
			Logger: logger,
		})
		if httpErr != nil {
			return fmt.Errorf("unable to create HTTP server: %w", httpErr)
		}
		shutdowns.RegisterCtx(httpSrv.Shutdown)

		// Register only ACME challenge routes on HTTP server
		routes.RegisterChallengeServer(httpSrv, svcs)

		httpListener, httpListenErr := net.Listen("tcp", fmt.Sprintf(":%d", cfg.HttpPort))
		if httpListenErr != nil {
			return fmt.Errorf("unable to create HTTP listener: %w", httpListenErr)
		}

		go func() {
			logger.Info("HTTP challenge server started", "addr", httpListener.Addr().String())
			serveErr := httpSrv.Serve(ctx, httpListener)
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
