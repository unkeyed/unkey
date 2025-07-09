package ctrl

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"log/slog"

	"github.com/unkeyed/unkey/go/apps/ctrl/services/ctrl"
	"github.com/unkeyed/unkey/go/apps/ctrl/services/version"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/builder"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hydra"
	"github.com/unkeyed/unkey/go/pkg/otel"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/shutdown"
	pkgversion "github.com/unkeyed/unkey/go/pkg/version"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

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

	// Initialize Hydra workflow engine with DSN
	hydraEngine, err := hydra.New(hydra.Config{
		DSN:        cfg.DatabaseHydra,
		Namespace:  "ctrl",
		Clock:      cfg.Clock,
		Logger:     logger,
		Marshaller: hydra.NewJSONMarshaller(),
	})
	if err != nil {
		return fmt.Errorf("failed to initialize hydra engine: %w", err)
	}

	// Create Hydra worker
	hydraWorker, err := hydra.NewWorker(hydraEngine, hydra.WorkerConfig{
		WorkerID:          cfg.InstanceID,
		Concurrency:       10,
		PollInterval:      2 * time.Second, // Less aggressive polling
		HeartbeatInterval: 30 * time.Second,
		ClaimTimeout:      30 * time.Minute, // Handle long builds
		CronInterval:      1 * time.Minute,  // Standard cron interval
	})
	if err != nil {
		return fmt.Errorf("unable to create hydra worker: %w", err)
	}

	// Create the mock builder service for demo
	builderService := builder.NewMockService()

	// Register deployment workflow with Hydra worker
	// TODO: Replace nil with actual metald client when available
	deployWorkflow := version.NewDeployWorkflow(database, logger, builderService, nil)
	err = hydra.RegisterWorkflow(hydraWorker, deployWorkflow)
	if err != nil {
		return fmt.Errorf("unable to register deployment workflow: %w", err)
	}

	// Create the service implementations
	ctrlSvc := ctrl.New(cfg.InstanceID, database)
	versionSvc := version.New(database, hydraEngine, builderService, logger)

	// Create the connect handler
	mux := http.NewServeMux()

	// Create the service handlers with interceptors
	ctrlPath, ctrlHandler := ctrlv1connect.NewCtrlServiceHandler(ctrlSvc)
	mux.Handle(ctrlPath, ctrlHandler)

	versionPath, versionHandler := ctrlv1connect.NewVersionServiceHandler(versionSvc)
	mux.Handle(versionPath, versionHandler)

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
		Addr:         addr,
		Handler:      h2cHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Register server shutdown
	shutdowns.RegisterCtx(server.Shutdown)

	// Start Hydra worker
	go func() {
		logger.Info("Starting Hydra workflow worker")
		if err := hydraWorker.Start(ctx); err != nil {
			logger.Error("Failed to start Hydra worker", "error", err)
		}
	}()
	shutdowns.RegisterCtx(hydraWorker.Shutdown)

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

	// Wait for signal and handle shutdown
	logger.Info("Ctrl server started successfully")
	if err := shutdowns.WaitForSignal(ctx); err != nil {
		logger.Error("Shutdown failed", "error", err)
		return err
	}

	logger.Info("Ctrl server shut down successfully")
	return nil
}
