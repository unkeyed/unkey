package ctrl

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	restate "github.com/restatedev/sdk-go"
	restateIngress "github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/prometheus"
	"github.com/unkeyed/unkey/pkg/shutdown"
	pkgversion "github.com/unkeyed/unkey/pkg/version"
	"github.com/unkeyed/unkey/svc/ctrl/services/acme"
	"github.com/unkeyed/unkey/svc/ctrl/services/build/backend/depot"
	"github.com/unkeyed/unkey/svc/ctrl/services/build/backend/docker"
	buildStorage "github.com/unkeyed/unkey/svc/ctrl/services/build/storage"
	"github.com/unkeyed/unkey/svc/ctrl/services/cluster"
	"github.com/unkeyed/unkey/svc/ctrl/services/ctrl"
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

	shutdowns := shutdown.New()

	if cfg.OtelEnabled {
		grafanaErr := otel.InitGrafana(ctx, otel.Config{
			Application:     "ctrl",
			Version:         pkgversion.Version,
			InstanceID:      cfg.InstanceID,
			CloudRegion:     cfg.Region,
			TraceSampleRate: cfg.OtelTraceSamplingRate,
		},
			shutdowns,
		)
		if grafanaErr != nil {
			return fmt.Errorf("unable to init grafana: %w", grafanaErr)
		}
	}

	logger := logging.New()
	if cfg.InstanceID != "" {
		logger = logger.With(slog.String("instanceID", cfg.InstanceID))
	}
	if cfg.Platform != "" {
		logger = logger.With(slog.String("platform", cfg.Platform))
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

	// Initialize database
	database, err := db.New(db.Config{
		PrimaryDSN:  cfg.DatabasePrimary,
		ReadOnlyDSN: "",
		Logger:      logger,
	})
	if err != nil {
		return fmt.Errorf("unable to create db: %w", err)
	}

	shutdowns.Register(database.Close)

	buildStorage, err := buildStorage.NewS3(buildStorage.S3Config{
		Logger:            logger,
		S3URL:             cfg.BuildS3.URL,
		S3PresignURL:      cfg.BuildS3.ExternalURL, // Empty for Depot, set for Docker
		S3Bucket:          cfg.BuildS3.Bucket,
		S3AccessKeyID:     cfg.BuildS3.AccessKeyID,
		S3AccessKeySecret: cfg.BuildS3.AccessKeySecret,
	})
	if err != nil {
		return fmt.Errorf("unable to create build storage: %w", err)
	}

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

	var buildService ctrlv1connect.BuildServiceClient
	switch cfg.BuildBackend {
	case BuildBackendDocker:
		buildService = docker.New(docker.Config{
			InstanceID:    cfg.InstanceID,
			DB:            database,
			Logger:        logger,
			BuildPlatform: docker.BuildPlatform(cfg.GetBuildPlatform()),
			Storage:       buildStorage,
		})
		logger.Info("Using Docker build backend", "presign_url", cfg.BuildS3.ExternalURL)

	case BuildBackendDepot:
		buildService = depot.New(depot.Config{
			InstanceID:     cfg.InstanceID,
			DB:             database,
			RegistryConfig: depot.RegistryConfig(cfg.GetRegistryConfig()),
			BuildPlatform:  depot.BuildPlatform(cfg.GetBuildPlatform()),
			DepotConfig:    depot.DepotConfig(cfg.GetDepotConfig()),
			Clickhouse:     ch,
			Logger:         logger,
			Storage:        buildStorage,
		})
		logger.Info("Using Depot build backend")

	default:
		return fmt.Errorf("unknown build backend: %s (must be 'docker' or 'depot')", cfg.BuildBackend)
	}

	// Restate ingress client for invoking workflows
	restateClientOpts := []restate.IngressClientOption{}
	if cfg.Restate.APIKey != "" {
		restateClientOpts = append(restateClientOpts, restate.WithAuthKey(cfg.Restate.APIKey))
	}
	restateClient := restateIngress.NewClient(cfg.Restate.URL, restateClientOpts...)

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

	// Health check endpoint for load balancers and orchestrators
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.Handle(ctrlv1connect.NewBuildServiceHandler(buildService))
	mux.Handle(ctrlv1connect.NewCtrlServiceHandler(ctrl.New(cfg.InstanceID, database)))
	mux.Handle(ctrlv1connect.NewDeploymentServiceHandler(deployment.New(deployment.Config{
		Database:         database,
		Restate:          restateClient,
		BuildService:     buildService,
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
	shutdowns.RegisterCtx(server.Shutdown)

	// Start server
	go func() {
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
			logger.Error("Server failed", "error", err)
		}
	}()

	if cfg.PrometheusPort > 0 {
		prom, promErr := prometheus.New(prometheus.Config{
			Logger: logger,
		})
		if promErr != nil {
			return fmt.Errorf("failed to create prometheus server: %w", promErr)
		}

		shutdowns.RegisterCtx(prom.Shutdown)
		ln, lnErr := net.Listen("tcp", fmt.Sprintf(":%d", cfg.PrometheusPort))
		if lnErr != nil {
			return fmt.Errorf("unable to listen on port %d: %w", cfg.PrometheusPort, lnErr)
		}
		go func() {
			logger.Info("prometheus started", "port", cfg.PrometheusPort)
			if serveErr := prom.Serve(ctx, ln); serveErr != nil {
				logger.Error("failed to start prometheus server", "error", serveErr)
			}
		}()
	}

	// Wait for signal and handle shutdown
	logger.Info("Ctrl server started successfully")
	if err := shutdowns.WaitForSignal(ctx); err != nil {
		logger.Error("Shutdown failed", "error", err)
		return err
	}

	logger.Info("Ctrl server shut down successfully")
	return nil
}
