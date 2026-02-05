package frontline

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/otel"
	"github.com/unkeyed/unkey/pkg/prometheus"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rpc/interceptor"
	"github.com/unkeyed/unkey/pkg/runner"
	pkgtls "github.com/unkeyed/unkey/pkg/tls"
	"github.com/unkeyed/unkey/pkg/version"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/routes"
	"github.com/unkeyed/unkey/svc/frontline/services/caches"
	"github.com/unkeyed/unkey/svc/frontline/services/certmanager"
	"github.com/unkeyed/unkey/svc/frontline/services/proxy"
	"github.com/unkeyed/unkey/svc/frontline/services/router"
)

// Run starts the frontline server
// frontline is our multi-tenant frontline service that:
// - Accepts requests from NLBs (<region>.aws.unkey.app)
// - Terminates TLS if the deployment exists in its region
// - Forwards requests to the per-tenant/environment sentinel service
// - OR forwards to another region if no local deployment exists
func Run(ctx context.Context, cfg Config) error {
	err := cfg.Validate()
	if err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	// Create cached clock with millisecond resolution for efficient time tracking
	clk := clock.New()

	// Initialize OTEL before creating logger so the logger picks up the OTLP handler
	var shutdownGrafana func(context.Context) error
	if cfg.OtelEnabled {
		shutdownGrafana, err = otel.InitGrafana(ctx, otel.Config{
			Application:     "frontline",
			Version:         version.Version,
			InstanceID:      cfg.FrontlineID,
			CloudRegion:     cfg.Region,
			TraceSampleRate: cfg.OtelTraceSamplingRate,
		})
		if err != nil {
			return fmt.Errorf("unable to init grafana: %w", err)
		}
	}

	// Configure global logger with base attributes
	if cfg.FrontlineID != "" {
		logger.AddBaseAttrs(slog.String("instanceID", cfg.FrontlineID))
	}

	if cfg.Region != "" {
		logger.AddBaseAttrs(slog.String("region", cfg.Region))
	}

	if version.Version != "" {
		logger.AddBaseAttrs(slog.String("version", version.Version))
	}

	r := runner.New()
	defer r.Recover()

	r.DeferCtx(shutdownGrafana)

	if cfg.PrometheusPort > 0 {
		prom, promErr := prometheus.New()
		if promErr != nil {
			return fmt.Errorf("unable to start prometheus: %w", promErr)
		}

		promListener, listenErr := net.Listen("tcp", fmt.Sprintf(":%d", cfg.PrometheusPort))
		if listenErr != nil {
			return fmt.Errorf("unable to listen on port %d: %w", cfg.PrometheusPort, listenErr)
		}

		r.DeferCtx(prom.Shutdown)
		r.Go(func(ctx context.Context) error {
			promListenErr := prom.Serve(ctx, promListener)
			if promListenErr != nil && !errors.Is(promListenErr, context.Canceled) {
				return fmt.Errorf("prometheus server failed: %w", promListenErr)
			}
			return nil
		})
	}

	var vaultClient vaultv1connect.VaultServiceClient
	if cfg.VaultURL != "" {
		vaultClient = vaultv1connect.NewVaultServiceClient(
			http.DefaultClient,
			cfg.VaultURL,
			connect.WithInterceptors(interceptor.NewHeaderInjector(map[string]string{
				"Authorization": "Bearer " + cfg.VaultToken,
			})),
		)
		logger.Info("Vault client initialized", "url", cfg.VaultURL)
	} else {
		logger.Warn("Vault not configured - TLS certificate decryption will be unavailable")
	}

	db, err := db.New(db.Config{
		PrimaryDSN:  cfg.DatabasePrimary,
		ReadOnlyDSN: cfg.DatabaseReadonlyReplica,
	})
	if err != nil {
		return fmt.Errorf("unable to create partitioned db: %w", err)
	}
	r.Defer(db.Close)

	// Initialize caches
	cache, err := caches.New(caches.Config{
		Clock: clk,
	})
	if err != nil {
		return fmt.Errorf("unable to create caches: %w", err)
	}

	// Initialize certificate manager for dynamic TLS
	var certManager certmanager.Service
	if vaultClient != nil {
		certManager = certmanager.New(certmanager.Config{
			DB:                  db,
			TLSCertificateCache: cache.TLSCertificates,
			Vault:               vaultClient,
		})
	}

	// Initialize router service
	routerSvc, err := router.New(router.Config{
		Region:                 cfg.Region,
		DB:                     db,
		FrontlineRouteCache:    cache.FrontlineRoutes,
		SentinelsByEnvironment: cache.SentinelsByEnvironment,
	})
	if err != nil {
		return fmt.Errorf("unable to create router service: %w", err)
	}

	// Initialize proxy service with shared transport for connection pooling
	// nolint:exhaustruct
	proxySvc, err := proxy.New(proxy.Config{
		FrontlineID: cfg.FrontlineID,
		Region:      cfg.Region,
		ApexDomain:  cfg.ApexDomain,
		Clock:       clk,
		MaxHops:     cfg.MaxHops,
		// Use defaults for transport settings (200 max idle conns, 90s timeout, etc.)
	})
	if err != nil {
		return fmt.Errorf("unable to create proxy service: %w", err)
	}

	// Create TLS config - either from static files (dev mode) or dynamic certificates (production)
	var tlsConfig *pkgtls.Config
	if cfg.EnableTLS {
		if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
			// Dev mode: static file-based certificate
			fileTLSConfig, tlsErr := pkgtls.NewFromFiles(cfg.TLSCertFile, cfg.TLSKeyFile)
			if tlsErr != nil {
				return fmt.Errorf("failed to load TLS certificate from files: %w", tlsErr)
			}
			tlsConfig = fileTLSConfig
			logger.Info("TLS configured with static certificate files",
				"certFile", cfg.TLSCertFile,
				"keyFile", cfg.TLSKeyFile)
		} else if certManager != nil {
			// Production mode: dynamic certificates from database/vault
			//nolint:exhaustruct
			tlsConfig = &tls.Config{
				GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
					return certManager.GetCertificate(context.Background(), hello.ServerName)
				},
				MinVersion: tls.VersionTLS12,
				// Enable session resumption for faster subsequent connections
				// Session tickets allow clients to skip the full TLS handshake
				SessionTicketsDisabled: false,
				// Let Go's TLS implementation choose optimal cipher suites
				// This prefers TLS 1.3 when available (1-RTT vs 2-RTT for TLS 1.2)
				PreferServerCipherSuites: false,
			}
			logger.Info("TLS configured with dynamic certificate manager")
		}
	}

	acmeClient := ctrlv1connect.NewAcmeServiceClient(ptr.P(http.Client{}), cfg.CtrlAddr)
	svcs := &routes.Services{
		Region:        cfg.Region,
		RouterService: routerSvc,
		ProxyService:  proxySvc,
		Clock:         clk,
		AcmeClient:    acmeClient,
	}

	// Start HTTPS frontline server (main proxy server)
	if cfg.HttpsPort > 0 {
		httpsSrv, httpsErr := zen.New(zen.Config{
			TLS: tlsConfig,
			// Use longer timeouts for proxy operations
			// WriteTimeout must be longer than the transport's ResponseHeaderTimeout (30s)
			// so that transport timeouts can be caught and handled properly in ErrorHandler
			ReadTimeout:        30 * time.Second,
			WriteTimeout:       60 * time.Second,
			Flags:              nil,
			EnableH2C:          false,
			MaxRequestBodySize: 0,
		})
		if httpsErr != nil {
			return fmt.Errorf("unable to create HTTPS server: %w", httpsErr)
		}
		r.RegisterHealth(httpsSrv.Mux(), "/_unkey/internal/health")
		r.DeferCtx(httpsSrv.Shutdown)

		// Register all frontline routes on HTTPS server
		routes.Register(httpsSrv, svcs)

		httpsListener, httpsListenErr := net.Listen("tcp", fmt.Sprintf(":%d", cfg.HttpsPort))
		if httpsListenErr != nil {
			return fmt.Errorf("unable to create HTTPS listener: %w", httpsListenErr)
		}

		r.Go(func(ctx context.Context) error {
			logger.Info("HTTPS frontline server started", "addr", httpsListener.Addr().String())
			serveErr := httpsSrv.Serve(ctx, httpsListener)
			if serveErr != nil && !errors.Is(serveErr, context.Canceled) {
				return fmt.Errorf("https server error: %w", serveErr)
			}
			return nil
		})
	}

	// Start HTTP challenge server (ACME only for Let's Encrypt)
	if cfg.HttpPort > 0 {
		httpSrv, httpErr := zen.New(zen.Config{
			TLS:                nil,
			Flags:              nil,
			EnableH2C:          false,
			MaxRequestBodySize: 0,
			ReadTimeout:        0,
			WriteTimeout:       0,
		})
		if httpErr != nil {
			return fmt.Errorf("unable to create HTTP server: %w", httpErr)
		}
		r.RegisterHealth(httpSrv.Mux(), "/_unkey/internal/health")
		r.DeferCtx(httpSrv.Shutdown)

		// Register only ACME challenge routes on HTTP server
		routes.RegisterChallengeServer(httpSrv, svcs)

		httpListener, httpListenErr := net.Listen("tcp", fmt.Sprintf(":%d", cfg.HttpPort))
		if httpListenErr != nil {
			return fmt.Errorf("unable to create HTTP listener: %w", httpListenErr)
		}

		r.Go(func(ctx context.Context) error {
			logger.Info("HTTP challenge server started", "addr", httpListener.Addr().String())
			serveErr := httpSrv.Serve(ctx, httpListener)
			if serveErr != nil && !errors.Is(serveErr, context.Canceled) {
				return fmt.Errorf("http server error: %w", serveErr)
			}
			return nil
		})
	}

	logger.Info("Frontline server initialized", "region", cfg.Region, "apexDomain", cfg.ApexDomain)

	// Wait for either OS signals or context cancellation, then shutdown
	if err := r.Wait(ctx); err != nil {
		return fmt.Errorf("shutdown failed: %w", err)
	}

	logger.Info("Frontline server shut down successfully")
	return nil
}
