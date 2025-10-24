package ctrl

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"
	restateIngress "github.com/restatedev/sdk-go/ingress"
	restateServer "github.com/restatedev/sdk-go/server"
	"github.com/unkeyed/unkey/go/apps/ctrl/middleware"
	"github.com/unkeyed/unkey/go/apps/ctrl/services/acme"
	"github.com/unkeyed/unkey/go/apps/ctrl/services/build/backend/depot"
	"github.com/unkeyed/unkey/go/apps/ctrl/services/build/backend/docker"
	buildStorage "github.com/unkeyed/unkey/go/apps/ctrl/services/build/storage"
	"github.com/unkeyed/unkey/go/apps/ctrl/services/ctrl"
	"github.com/unkeyed/unkey/go/apps/ctrl/services/deployment"
	"github.com/unkeyed/unkey/go/apps/ctrl/services/openapi"
	"github.com/unkeyed/unkey/go/apps/ctrl/workflows/certificate"
	"github.com/unkeyed/unkey/go/apps/ctrl/workflows/deploy"
	"github.com/unkeyed/unkey/go/apps/ctrl/workflows/routing"
	deployTLS "github.com/unkeyed/unkey/go/deploy/pkg/tls"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/go/gen/proto/krane/v1/kranev1connect"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/retry"
	"github.com/unkeyed/unkey/go/pkg/shutdown"
	"github.com/unkeyed/unkey/go/pkg/vault"
	"github.com/unkeyed/unkey/go/pkg/vault/storage"
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

	var vaultSvc *vault.Service
	if len(cfg.VaultMasterKeys) > 0 {
		vaultStorage, err := storage.NewS3(storage.S3Config{
			Logger:            logger,
			S3URL:             cfg.VaultS3.URL,
			S3Bucket:          cfg.VaultS3.Bucket,
			S3AccessKeyID:     cfg.VaultS3.AccessKeyID,
			S3AccessKeySecret: cfg.VaultS3.AccessKeySecret,
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
	// make go happy
	_ = vaultSvc

	// Initialize database
	database, err := db.New(db.Config{
		PrimaryDSN:  cfg.DatabasePrimary,
		ReadOnlyDSN: "",
		Logger:      logger,
	})
	if err != nil {
		return fmt.Errorf("unable to create db: %w", err)
	}

	partitionDB, err := db.New(db.Config{
		PrimaryDSN:  cfg.DatabasePartition,
		ReadOnlyDSN: "",
		Logger:      logger,
	})
	if err != nil {
		return fmt.Errorf("unable to create partition db: %w", err)
	}

	shutdowns.Register(partitionDB.Close)
	shutdowns.Register(database.Close)

	// Create krane client for VM operations
	var httpClient *http.Client
	var authMode string

	if cfg.SPIFFESocketPath != "" {
		// Use SPIRE authentication when socket path is provided
		tlsConfig := deployTLS.Config{
			Mode:              deployTLS.ModeSPIFFE,
			SPIFFESocketPath:  cfg.SPIFFESocketPath,
			CertFile:          "",
			KeyFile:           "",
			CAFile:            "",
			SPIFFETimeout:     "",
			EnableCertCaching: false,
			CertCacheTTL:      0,
		}

		tlsProvider, tlsErr := deployTLS.NewProvider(ctx, tlsConfig)
		if tlsErr != nil {
			return fmt.Errorf("failed to create TLS provider for krane: %w", tlsErr)
		}

		httpClient = tlsProvider.HTTPClient()
		authMode = "SPIRE"
	} else {
		// Fall back to plain HTTP for local development
		httpClient = &http.Client{}
		authMode = "plain HTTP"
	}

	httpClient.Timeout = 30 * time.Second

	kraneClient := kranev1connect.NewDeploymentServiceClient(
		httpClient,
		cfg.KraneAddress,
		connect.WithInterceptors(connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
			return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				req.Header().Set("Authorization", "Bearer dev_user_ctrl")
				return next(ctx, req)
			}
		})),
	)
	logger.Info("krane client configured", "address", cfg.KraneAddress, "auth_mode", authMode)

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

	var buildService ctrlv1connect.BuildServiceClient
	switch cfg.BuildBackend {
	case BuildBackendDocker:
		buildService = docker.New(docker.Config{
			DB:      database,
			Logger:  logger,
			Storage: buildStorage,
		})
		logger.Info("Using Docker build backend", "presign_url", cfg.BuildS3.ExternalURL)

	case BuildBackendDepot:
		buildService = depot.New(depot.Config{
			InstanceID:    cfg.InstanceID,
			DB:            database,
			APIUrl:        cfg.Depot.APIUrl,
			RegistryUrl:   cfg.Depot.RegistryUrl,
			Username:      cfg.Depot.Username,
			AccessToken:   cfg.Depot.AccessToken,
			BuildPlatform: cfg.Depot.BuildPlatform,
			ProjectRegion: cfg.Depot.ProjectRegion,
			Logger:        logger,
			Storage:       buildStorage,
		})
		logger.Info("Using Depot build backend")

	default:
		return fmt.Errorf("unknown build backend: %s (must be 'docker' or 'depot')", cfg.BuildBackend)
	}

	// Restate Client and Server
	restateClient := restateIngress.NewClient(cfg.Restate.IngressURL)
	restateSrv := restateServer.NewRestate()

	restateSrv.Bind(hydrav1.NewDeploymentServiceServer(deploy.New(deploy.Config{
		Logger:        logger,
		DB:            database,
		PartitionDB:   partitionDB,
		Krane:         kraneClient,
		BuildClient:   buildService,
		DefaultDomain: cfg.DefaultDomain,
	})))

	restateSrv.Bind(hydrav1.NewRoutingServiceServer(routing.New(routing.Config{
		Logger:        logger,
		DB:            database,
		PartitionDB:   partitionDB,
		DefaultDomain: cfg.DefaultDomain,
	})))

	restateSrv.Bind(hydrav1.NewCertificateServiceServer(certificate.New(certificate.Config{
		Logger:      logger,
		DB:          database,
		PartitionDB: partitionDB,
		Vault:       vaultSvc,
	})))

	go func() {
		addr := fmt.Sprintf(":%d", cfg.Restate.HttpPort)
		logger.Info("Starting Restate server", "addr", addr)
		if startErr := restateSrv.Start(ctx, addr); startErr != nil {
			logger.Error("failed to start restate server", "error", startErr.Error())
		}
	}()

	// Register with Restate admin API if RegisterAs is configured
	if cfg.Restate.RegisterAs != "" {
		go func() {
			// Wait a moment for the restate server to be ready
			time.Sleep(2 * time.Second)

			registerURL := fmt.Sprintf("%s/deployments", cfg.Restate.AdminURL)
			payload := fmt.Sprintf(`{"uri": "%s"}`, cfg.Restate.RegisterAs)

			logger.Info("Registering with Restate", "admin_url", registerURL, "service_uri", cfg.Restate.RegisterAs)

			retrier := retry.New(
				retry.Attempts(10),
				retry.Backoff(func(n int) time.Duration {
					return 5 * time.Second
				}),
			)

			err := retrier.Do(func() error {
				req, err := http.NewRequestWithContext(ctx, "POST", registerURL, bytes.NewBufferString(payload))
				if err != nil {
					return fmt.Errorf("failed to create registration request: %w", err)
				}

				req.Header.Set("Content-Type", "application/json")

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return fmt.Errorf("failed to register with Restate: %w", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					return nil
				}

				return fmt.Errorf("registration returned status %d", resp.StatusCode)
			})

			if err != nil {
				logger.Error("failed to register with Restate after retries", "error", err.Error())
			} else {
				logger.Info("Successfully registered with Restate")
			}
		}()
	}

	// Create the connect handler
	mux := http.NewServeMux()

	// Create authentication middleware (required except for health check and ACME routes)
	authMiddleware := middleware.NewAuthMiddleware(middleware.AuthConfig{
		APIKey: cfg.APIKey,
	})
	authInterceptor := authMiddleware.ConnectInterceptor()

	if cfg.APIKey != "" {
		logger.Info("API key authentication enabled for ctrl service")
	} else {
		logger.Warn("No API key configured - authentication will reject all requests except health check and ACME routes")
	}

	// Create the service handlers with auth interceptor (always applied)
	connectOptions := []connect.HandlerOption{
		connect.WithInterceptors(authInterceptor),
	}
	mux.Handle(ctrlv1connect.NewBuildServiceHandler(buildService, connectOptions...))
	mux.Handle(ctrlv1connect.NewCtrlServiceHandler(ctrl.New(cfg.InstanceID, database), connectOptions...))
	mux.Handle(ctrlv1connect.NewDeploymentServiceHandler(deployment.New(deployment.Config{
		Database:     database,
		PartitionDB:  partitionDB,
		Restate:      restateClient,
		BuildService: buildService,
		Logger:       logger,
	}), connectOptions...))
	mux.Handle(ctrlv1connect.NewOpenApiServiceHandler(openapi.New(database, logger), connectOptions...))
	mux.Handle(ctrlv1connect.NewAcmeServiceHandler(acme.New(acme.Config{
		PartitionDB: partitionDB,
		DB:          database,
		Logger:      logger,
	}), connectOptions...))

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
