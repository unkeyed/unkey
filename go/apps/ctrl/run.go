package ctrl

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/apps/ctrl/services/acme"
	"github.com/unkeyed/unkey/go/apps/ctrl/services/acme/providers"
	"github.com/unkeyed/unkey/go/apps/ctrl/services/ctrl"
	"github.com/unkeyed/unkey/go/apps/ctrl/services/deployment"
	"github.com/unkeyed/unkey/go/apps/ctrl/services/openapi"
	deployTLS "github.com/unkeyed/unkey/go/deploy/pkg/tls"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/gen/proto/metald/v1/metaldv1connect"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hydra"
	"github.com/unkeyed/unkey/go/pkg/otel"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/shutdown"
	"github.com/unkeyed/unkey/go/pkg/uid"
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

	// Create metald client for VM operations
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
			return fmt.Errorf("failed to create TLS provider for metald: %w", tlsErr)
		}

		httpClient = tlsProvider.HTTPClient()
		authMode = "SPIRE"
	} else {
		// Fall back to plain HTTP for local development
		httpClient = &http.Client{}
		authMode = "plain HTTP"
	}

	httpClient.Timeout = 30 * time.Second

	metaldClient := metaldv1connect.NewVmServiceClient(
		httpClient,
		cfg.MetaldAddress,
		connect.WithInterceptors(connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
			return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				logger.Info("Adding auth headers to metald request", "procedure", req.Spec().Procedure)
				req.Header().Set("Authorization", "Bearer dev_user_ctrl")
				req.Header().Set("X-Tenant-ID", "ctrl-tenant")
				return next(ctx, req)
			}
		})),
	)
	logger.Info("metald client configured", "address", cfg.MetaldAddress, "auth_mode", authMode)

	// Register deployment workflow with Hydra worker
	deployWorkflow := deployment.NewDeployWorkflow(deployment.DeployWorkflowConfig{
		Logger:        logger,
		DB:            database,
		PartitionDB:   partitionDB,
		MetaldBackend: cfg.MetaldBackend,
		MetalD:        metaldClient,
		DefaultDomain: cfg.DefaultDomain,
	})
	err = hydra.RegisterWorkflow(hydraWorker, deployWorkflow)
	if err != nil {
		return fmt.Errorf("unable to register deployment workflow: %w", err)
	}

	// Create the connect handler
	mux := http.NewServeMux()

	// Create the service handlers with interceptors
	mux.Handle(ctrlv1connect.NewCtrlServiceHandler(ctrl.New(cfg.InstanceID, database)))
	mux.Handle(ctrlv1connect.NewDeploymentServiceHandler(deployment.New(database, partitionDB, hydraEngine, logger)))
	mux.Handle(ctrlv1connect.NewOpenApiServiceHandler(openapi.New(database, logger)))
	mux.Handle(ctrlv1connect.NewAcmeServiceHandler(acme.New(acme.Config{
		PartitionDB: partitionDB,
		DB:          database,
		HydraEngine: hydraEngine,
		Logger:      logger,
	})))

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

	if cfg.Acme.Enabled {
		acmeClient, err := acme.GetOrCreateUser(ctx, acme.UserConfig{
			DB:          database,
			Logger:      logger,
			Vault:       vaultSvc,
			WorkspaceID: "unkey",
		})
		if err != nil {
			return fmt.Errorf("failed to create ACME user: %w", err)
		}

		// Set up our custom HTTP-01 challenge provider on the ACME client
		httpProvider := providers.NewHTTPProvider(providers.HTTPProviderConfig{
			DB:     database,
			Logger: logger,
		})
		err = acmeClient.Challenge.SetHTTP01Provider(httpProvider)
		if err != nil {
			logger.Error("failed to set HTTP-01 provider", "error", err)
			return fmt.Errorf("failed to set HTTP-01 provider: %w", err)
		}

		// Set up Cloudflare DNS-01 challenge provider if enabled
		if cfg.Acme.Cloudflare.Enabled {
			cloudflareProvider, err := providers.NewCloudflareProvider(providers.CloudflareProviderConfig{
				DB:            database,
				Logger:        logger,
				APIToken:      cfg.Acme.Cloudflare.ApiToken,
				DefaultDomain: cfg.DefaultDomain,
			})
			if err != nil {
				logger.Error("failed to create Cloudflare DNS provider", "error", err)
				return fmt.Errorf("failed to create Cloudflare DNS provider: %w", err)
			}

			err = acmeClient.Challenge.SetDNS01Provider(cloudflareProvider)
			if err != nil {
				logger.Error("failed to set DNS-01 provider", "error", err)
				return fmt.Errorf("failed to set DNS-01 provider: %w", err)
			}

			logger.Info("Cloudflare DNS-01 challenge provider configured")

			if cfg.DefaultDomain != "" {
				wildcardDomain := "*." + cfg.DefaultDomain

				// Check if we already have a challenge or certificate for the wildcard domain
				_, err := db.Query.FindDomainByDomain(ctx, database.RO(), wildcardDomain)
				if err != nil && !db.IsNotFound(err) {
					logger.Error("Failed to check existing wildcard domain", "error", err, "domain", wildcardDomain)
				} else if db.IsNotFound(err) {
					now := time.Now().UnixMilli()
					domainID := uid.New("domain")

					// Insert domain record
					err = db.Query.InsertDomain(ctx, database.RW(), db.InsertDomainParams{
						ID:          domainID,
						WorkspaceID: "unkey", // Default workspace for wildcard cert
						Domain:      wildcardDomain,
						CreatedAt:   now,
						UpdatedAt:   sql.NullInt64{Valid: true, Int64: now},
						Type:        db.DomainsTypeCustom,
					})
					if err != nil {
						logger.Error("Failed to create wildcard domain", "error", err, "domain", wildcardDomain)
					} else {
						// Insert challenge record
						expiresAt := time.Now().Add(90 * 24 * time.Hour).UnixMilli() // 90 days

						err = db.Query.InsertAcmeChallenge(ctx, database.RW(), db.InsertAcmeChallengeParams{
							WorkspaceID:   "unkey",
							DomainID:      domainID,
							Token:         "",
							Authorization: "",
							Status:        db.AcmeChallengesStatusWaiting,
							Type:          db.AcmeChallengesTypeDNS01, // Use DNS-01 for wildcard
							CreatedAt:     now,
							UpdatedAt:     sql.NullInt64{Valid: true, Int64: now},
							ExpiresAt:     expiresAt,
						})
						if err != nil {
							logger.Error("Failed to create wildcard challenge", "error", err, "domain", wildcardDomain)
						} else {
							logger.Info("Created wildcard domain and challenge", "domain", wildcardDomain)
						}
					}
				}
			}
		}

		// Register deployment workflow with Hydra worker
		acmeWorkflows := acme.NewCertificateChallenge(acme.CertificateChallengeConfig{
			DB:          database,
			PartitionDB: partitionDB,
			Logger:      logger,
			AcmeClient:  acmeClient,
			Vault:       vaultSvc,
		})
		err = hydra.RegisterWorkflow(hydraWorker, acmeWorkflows)
		if err != nil {
			logger.Error("unable to register ACME certificate workflow", "error", err)
			return fmt.Errorf("unable to register deployment workflow: %w", err)
		}

		go func() {
			logger.Info("Starting cert worker")

			// HTTP-01 challenges are always available (we always set up HTTP provider)
			supportedTypes := []db.AcmeChallengesType{db.AcmeChallengesTypeHTTP01}

			// DNS-01 challenges require Cloudflare to be enabled
			if cfg.Acme.Cloudflare.Enabled {
				supportedTypes = append(supportedTypes, db.AcmeChallengesTypeDNS01)
			}

			registerErr := hydraEngine.RegisterCron("*/5 * * * *", "start-certificate-challenges", func(ctx context.Context, payload hydra.CronPayload) error {
				executableChallenges, err := db.Query.ListExecutableChallenges(ctx, database.RO(), supportedTypes)
				if err != nil {
					logger.Error("Failed to start workflow", "error", err)
					return err
				}

				logger.Info("Starting certificate challenges",
					"executable_challenges", len(executableChallenges),
					"supported_types", supportedTypes)

				for _, challenge := range executableChallenges {
					executionID, err := hydraEngine.StartWorkflow(ctx, "certificate_challenge",
						acme.CertificateChallengeRequest{
							WorkspaceID: challenge.WorkspaceID,
							Domain:      challenge.Domain,
						},
						hydra.WithMaxAttempts(24),
						hydra.WithTimeout(25*time.Hour),
						hydra.WithRetryBackoff(1*time.Hour),
					)
					if err != nil {
						logger.Error("Failed to start workflow", "error", err)
						continue
					}

					logger.Info("Workflow started", "executionID", executionID)
				}

				return nil
			})

			if registerErr != nil {
				logger.Error("Failed to register daily report cron job", "error", err)
				return
			}
		}()
	}

	// Start Hydra worker
	go func() {
		logger.Info("Starting Hydra workflow worker")
		if err := hydraWorker.Start(ctx); err != nil {
			logger.Error("Failed to start Hydra worker", "error", err)
		}
	}()

	shutdowns.RegisterCtx(hydraWorker.Shutdown)

	// Wait for signal and handle shutdown
	logger.Info("Ctrl server started successfully")
	if err := shutdowns.WaitForSignal(ctx); err != nil {
		logger.Error("Shutdown failed", "error", err)
		return err
	}

	logger.Info("Ctrl server shut down successfully")
	return nil
}
