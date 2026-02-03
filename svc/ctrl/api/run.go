package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	restate "github.com/restatedev/sdk-go"
	restateIngress "github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/prometheus"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/pkg/runner"
	pkgversion "github.com/unkeyed/unkey/pkg/version"
	"github.com/unkeyed/unkey/svc/ctrl/services/acme"
	"github.com/unkeyed/unkey/svc/ctrl/services/cluster"
	"github.com/unkeyed/unkey/svc/ctrl/services/ctrl"
	"github.com/unkeyed/unkey/svc/ctrl/services/customdomain"
	"github.com/unkeyed/unkey/svc/ctrl/services/deployment"
	"github.com/unkeyed/unkey/svc/ctrl/services/openapi"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// Run starts the control plane server with the provided configuration.
//
// This function initializes all required services and starts the HTTP/2 Connect server.
// It performs these major initialization steps:
// 1. Validates configuration and initializes structured logging
// 2. Sets up OpenTelemetry if enabled
// 3. Initializes database and build storage
// 4. Creates Restate ingress client for invoking workflows
// 5. Starts HTTP/2 server with all Connect handlers
//
// The server handles graceful shutdown when context is cancelled, properly
// closing all services and database connections.
//
// Returns an error if configuration validation fails, service initialization
// fails, or during server startup. Context cancellation results in
// clean shutdown with nil error.
func Run(ctx context.Context, cfg Config) error {
	err := cfg.Validate()
	if err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

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

	if cfg.TLSConfig != nil {
		logger.Info("TLS is enabled, server will use HTTPS")
	}

	r := runner.New(logger)
	defer r.Recover()

	if cfg.OtelEnabled {
		grafanaErr := otel.InitGrafana(ctx, otel.Config{
			Application:     "ctrl",
			Version:         pkgversion.Version,
			InstanceID:      cfg.InstanceID,
			CloudRegion:     cfg.Region,
			TraceSampleRate: cfg.OtelTraceSamplingRate,
		},
			r,
		)
		if grafanaErr != nil {
			return fmt.Errorf("unable to init grafana: %w", grafanaErr)
		}
	}

	// Initialize database
	database, err := db.New(db.Config{
		PrimaryDSN:  cfg.DatabasePrimary,
		ReadOnlyDSN: "",
		Logger:      logger,
	})
	if err != nil {
		return fmt.Errorf("unable to create db: %w", err)
	}

	r.Defer(database.Close)

	// Restate ingress client for invoking workflows
	restateClientOpts := []restate.IngressClientOption{}
	if cfg.Restate.APIKey != "" {
		restateClientOpts = append(restateClientOpts, restate.WithAuthKey(cfg.Restate.APIKey))
	}
	restateClient := restateIngress.NewClient(cfg.Restate.URL, restateClientOpts...)

	// Restate admin client for managing invocations
	restateAdminClient := restateadmin.New(restateadmin.Config{
		BaseURL: cfg.Restate.AdminURL,
		APIKey:  cfg.Restate.APIKey,
	})

	c := cluster.New(cluster.Config{
		Database: database,
		Logger:   logger,
		Bearer:   cfg.AuthToken,
	})

	// Initialize caches for ACME service (needed for certificate verification endpoint)
	clk := clock.New()
	domainCache, err := cache.New(cache.Config[string, db.CustomDomain]{
		Fresh:    5 * time.Minute,
		Stale:    10 * time.Minute,
		MaxSize:  10000,
		Logger:   logger,
		Resource: "domains",
		Clock:    clk,
	})
	if err != nil {
		return fmt.Errorf("failed to create domain cache: %w", err)
	}

	challengeCache, err := cache.New(cache.Config[string, db.AcmeChallenge]{
		Fresh:    10 * time.Second,
		Stale:    30 * time.Second,
		MaxSize:  1000,
		Logger:   logger,
		Resource: "acme_challenges",
		Clock:    clk,
	})
	if err != nil {
		return fmt.Errorf("failed to create challenge cache: %w", err)
	}

	// Create the connect handler
	mux := http.NewServeMux()

	r.RegisterHealth(mux)

	mux.Handle(ctrlv1connect.NewCtrlServiceHandler(ctrl.New(cfg.InstanceID, database)))
	mux.Handle(ctrlv1connect.NewDeploymentServiceHandler(deployment.New(deployment.Config{
		Database:         database,
		Restate:          restateClient,
		Logger:           logger,
		AvailableRegions: cfg.AvailableRegions,
	})))

	mux.Handle(ctrlv1connect.NewOpenApiServiceHandler(openapi.New(database, logger)))
	mux.Handle(ctrlv1connect.NewAcmeServiceHandler(acme.New(acme.Config{
		DB:             database,
		Logger:         logger,
		DomainCache:    domainCache,
		ChallengeCache: challengeCache,
	})))
	mux.Handle(ctrlv1connect.NewClusterServiceHandler(c))
	mux.Handle(ctrlv1connect.NewCustomDomainServiceHandler(customdomain.New(customdomain.Config{
		Database:     database,
		Restate:      restateClient,
		RestateAdmin: restateAdminClient,
		Logger:       logger,
		CnameDomain:  cfg.CnameDomain,
	})))

	if cfg.GitHubWebhookSecret != "" {
		mux.Handle("POST /webhooks/github", &GitHubWebhook{
			db:            database,
			logger:        logger,
			restate:       restateClient,
			webhookSecret: cfg.GitHubWebhookSecret,
		})
		logger.Info("GitHub webhook handler registered")
	} else {
		logger.Info("GitHub webhook handler not registered, no webhook secret configured")
	}

	// Configure server
	addr := fmt.Sprintf(":%d", cfg.HttpPort)

	// Use h2c for HTTP/2 without TLS (for development)
	h2cHandler := h2c.NewHandler(mux, &http2.Server{
		MaxHandlers:                  0,
		MaxConcurrentStreams:         0,
		MaxDecoderHeaderTableSize:    0,
		MaxEncoderHeaderTableSize:    0,
		MaxReadFrameSize:             0,
		PermitProhibitedCipherSuites: false,
		IdleTimeout:                  0,
		ReadIdleTimeout:              0,
		PingTimeout:                  0,
		WriteByteTimeout:             0,
		MaxUploadBufferPerConnection: 0,
		MaxUploadBufferPerStream:     0,
		NewWriteScheduler:            nil,
		CountError:                   nil,
	})

	server := &http.Server{
		Addr:              addr,
		Handler:           h2cHandler,
		ReadHeaderTimeout: 30 * time.Second,
		// Do not set timeouts here, our streaming rpcs will get canceled too frequently
	}

	// Register server shutdown
	r.DeferCtx(server.Shutdown)

	// Start server
	r.Go(func(ctx context.Context) error {
		logger.Info("Starting ctrl server", "addr", addr, "tls", cfg.TLSConfig != nil)

		var err error
		if cfg.TLSConfig != nil {
			server.TLSConfig = cfg.TLSConfig
			// For TLS, use the regular mux without h2c wrapper
			server.Handler = mux
			err = server.ListenAndServeTLS("", "")
		} else {
			err = server.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server failed: %w", err)
		}
		return nil
	})

	// Bootstrap certificates (wildcard domain records)
	if cfg.DefaultDomain != "" {
		certBootstrap := &certificateBootstrap{
			logger:         logger,
			database:       database,
			defaultDomain:  cfg.DefaultDomain,
			regionalDomain: cfg.RegionalDomain,
			regions:        cfg.AvailableRegions,
		}
		go certBootstrap.run(ctx)
	}

	if cfg.PrometheusPort > 0 {
		prom, promErr := prometheus.New(prometheus.Config{
			Logger: logger,
		})
		if promErr != nil {
			return fmt.Errorf("failed to create prometheus server: %w", promErr)
		}

		ln, lnErr := net.Listen("tcp", fmt.Sprintf(":%d", cfg.PrometheusPort))
		if lnErr != nil {
			return fmt.Errorf("unable to listen on port %d: %w", cfg.PrometheusPort, lnErr)
		}

		r.DeferCtx(prom.Shutdown)
		r.Go(func(ctx context.Context) error {
			logger.Info("prometheus started", "port", cfg.PrometheusPort)
			if serveErr := prom.Serve(ctx, ln); serveErr != nil && !errors.Is(serveErr, context.Canceled) {
				return fmt.Errorf("failed to start prometheus server: %w", serveErr)
			}
			return nil
		})
	}

	// Wait for signal and handle shutdown
	logger.Info("Ctrl server started successfully")
	if err := r.Wait(ctx); err != nil {
		logger.Error("Shutdown failed", "error", err)
		return err
	}

	logger.Info("Ctrl server shut down successfully")
	return nil
}
