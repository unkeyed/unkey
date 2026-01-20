package ctrl

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/go-acme/lego/v4/challenge"
	restate "github.com/restatedev/sdk-go"
	restateIngress "github.com/restatedev/sdk-go/ingress"
	restateServer "github.com/restatedev/sdk-go/server"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/prometheus"
	"github.com/unkeyed/unkey/pkg/retry"
	"github.com/unkeyed/unkey/pkg/shutdown"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/vault"
	"github.com/unkeyed/unkey/pkg/vault/storage"
	pkgversion "github.com/unkeyed/unkey/pkg/version"
	ctrlCaches "github.com/unkeyed/unkey/svc/ctrl/internal/caches"
	"github.com/unkeyed/unkey/svc/ctrl/services/acme"
	"github.com/unkeyed/unkey/svc/ctrl/services/acme/providers"
	"github.com/unkeyed/unkey/svc/ctrl/services/build/backend/depot"
	"github.com/unkeyed/unkey/svc/ctrl/services/build/backend/docker"
	buildStorage "github.com/unkeyed/unkey/svc/ctrl/services/build/storage"
	"github.com/unkeyed/unkey/svc/ctrl/services/cluster"
	"github.com/unkeyed/unkey/svc/ctrl/services/ctrl"
	"github.com/unkeyed/unkey/svc/ctrl/services/deployment"
	"github.com/unkeyed/unkey/svc/ctrl/services/openapi"
	"github.com/unkeyed/unkey/svc/ctrl/workflows/certificate"
	"github.com/unkeyed/unkey/svc/ctrl/workflows/deploy"
	"github.com/unkeyed/unkey/svc/ctrl/workflows/routing"
	"github.com/unkeyed/unkey/svc/ctrl/workflows/versioning"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// Run starts the control plane server with the provided configuration.
//
// This function initializes all required services and starts the HTTP/2 Connect server.
// It performs these major initialization steps:
// 1. Validates configuration and initializes structured logging
// 2. Sets up OpenTelemetry if enabled
// 3. Creates vault services for secrets and ACME certificates
// 4. Initializes database and build storage
// 5. Starts Restate workflow engine with service bindings
// 6. Configures ACME challenge providers (HTTP-01, DNS-01)
// 7. Registers with Restate admin API for service discovery
// 8. Starts HTTP/2 server with all Connect handlers
// 9. Boots up cluster management and starts certificate renewal
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

	// Disable CNAME following in lego to prevent it from following wildcard CNAMEs
	// (e.g., *.example.com -> loadbalancer.aws.com) and failing Route53 zone lookup.
	// Must be set before creating any ACME DNS providers.
	_ = os.Setenv("LEGO_DISABLE_CNAME_SUPPORT", "true")

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

	// Create vault service for general secrets (env vars, API keys, etc.)
	var vaultSvc *vault.Service
	if len(cfg.VaultMasterKeys) > 0 && cfg.VaultS3.URL != "" {
		vaultStorage, vaultStorageErr := storage.NewS3(storage.S3Config{
			Logger:            logger,
			S3URL:             cfg.VaultS3.URL,
			S3Bucket:          cfg.VaultS3.Bucket,
			S3AccessKeyID:     cfg.VaultS3.AccessKeyID,
			S3AccessKeySecret: cfg.VaultS3.AccessKeySecret,
		})
		if vaultStorageErr != nil {
			return fmt.Errorf("unable to create vault storage: %w", vaultStorageErr)
		}

		vaultSvc, err = vault.New(vault.Config{
			Logger:     logger,
			Storage:    vaultStorage,
			MasterKeys: cfg.VaultMasterKeys,
		})
		if err != nil {
			return fmt.Errorf("unable to create vault service: %w", err)
		}
		logger.Info("Vault service initialized", "bucket", cfg.VaultS3.Bucket)
	}

	// Create separate vault service for ACME certificates
	var acmeVaultSvc *vault.Service
	if len(cfg.AcmeVaultMasterKeys) > 0 && cfg.AcmeVaultS3.URL != "" {
		acmeVaultStorage, acmeStorageErr := storage.NewS3(storage.S3Config{
			Logger:            logger,
			S3URL:             cfg.AcmeVaultS3.URL,
			S3Bucket:          cfg.AcmeVaultS3.Bucket,
			S3AccessKeyID:     cfg.AcmeVaultS3.AccessKeyID,
			S3AccessKeySecret: cfg.AcmeVaultS3.AccessKeySecret,
		})
		if acmeStorageErr != nil {
			return fmt.Errorf("unable to create ACME vault storage: %w", acmeStorageErr)
		}

		acmeVaultSvc, err = vault.New(vault.Config{
			Logger:     logger,
			Storage:    acmeVaultStorage,
			MasterKeys: cfg.AcmeVaultMasterKeys,
		})
		if err != nil {
			return fmt.Errorf("unable to create ACME vault service: %w", err)
		}
		logger.Info("ACME vault service initialized", "bucket", cfg.AcmeVaultS3.Bucket)
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

	// Restate Client and Server
	restateClientOpts := []restate.IngressClientOption{}
	if cfg.Restate.APIKey != "" {
		restateClientOpts = append(restateClientOpts, restate.WithAuthKey(cfg.Restate.APIKey))
	}
	restateClient := restateIngress.NewClient(cfg.Restate.URL, restateClientOpts...)
	restateSrv := restateServer.NewRestate()

	c := cluster.New(cluster.Config{
		Database: database,
		Logger:   logger,
		Bearer:   cfg.AuthToken,
	})

	restateSrv.Bind(hydrav1.NewDeploymentServiceServer(deploy.New(deploy.Config{
		Logger:           logger,
		DB:               database,
		BuildClient:      buildService,
		DefaultDomain:    cfg.DefaultDomain,
		Vault:            vaultSvc,
		Cluster:          c,
		SentinelImage:    cfg.SentinelImage,
		AvailableRegions: cfg.AvailableRegions,
		Bearer:           cfg.AuthToken,
	})))

	restateSrv.Bind(hydrav1.NewRoutingServiceServer(routing.New(routing.Config{
		Logger:        logger,
		DB:            database,
		DefaultDomain: cfg.DefaultDomain,
	}), restate.WithIngressPrivate(true)))

	restateSrv.Bind(hydrav1.NewVersioningServiceServer(versioning.New(), restate.WithIngressPrivate(true)))

	// Initialize shared caches for ACME (needed for verification endpoint regardless of provider config)
	caches, cacheErr := ctrlCaches.New(ctrlCaches.Config{
		Logger: logger,
		Clock:  clock.New(),
	})
	if cacheErr != nil {
		return fmt.Errorf("failed to create ACME caches: %w", cacheErr)
	}

	// Setup ACME challenge providers
	var dnsProvider challenge.Provider
	var httpProvider challenge.Provider
	if cfg.Acme.Enabled {
		// HTTP-01 provider for regular (non-wildcard) domains
		httpProv, httpErr := providers.NewHTTPProvider(providers.HTTPConfig{
			DB:          database,
			Logger:      logger,
			DomainCache: caches.Domains,
		})
		if httpErr != nil {
			return fmt.Errorf("failed to create HTTP-01 provider: %w", httpErr)
		}
		httpProvider = httpProv
		logger.Info("ACME HTTP-01 provider enabled")

		// DNS-01 provider for wildcard domains (requires DNS provider config)
		if cfg.Acme.Cloudflare.Enabled {
			cfProvider, cfErr := providers.NewCloudflareProvider(providers.CloudflareConfig{
				DB:          database,
				Logger:      logger,
				APIToken:    cfg.Acme.Cloudflare.ApiToken,
				DomainCache: caches.Domains,
			})
			if cfErr != nil {
				return fmt.Errorf("failed to create Cloudflare DNS provider: %w", cfErr)
			}
			dnsProvider = cfProvider
			logger.Info("ACME Cloudflare DNS-01 provider enabled for wildcard certs")
		} else if cfg.Acme.Route53.Enabled {
			r53Provider, r53Err := providers.NewRoute53Provider(providers.Route53Config{
				DB:              database,
				Logger:          logger,
				AccessKeyID:     cfg.Acme.Route53.AccessKeyID,
				SecretAccessKey: cfg.Acme.Route53.SecretAccessKey,
				Region:          cfg.Acme.Route53.Region,
				HostedZoneID:    cfg.Acme.Route53.HostedZoneID,
				DomainCache:     caches.Domains,
			})
			if r53Err != nil {
				return fmt.Errorf("failed to create Route53 DNS provider: %w", r53Err)
			}
			dnsProvider = r53Provider
			logger.Info("ACME Route53 DNS-01 provider enabled for wildcard certs")
		}
	}

	// Certificate service needs a longer timeout for ACME DNS-01 challenges
	// which can take 5-10 minutes for DNS propagation
	restateSrv.Bind(hydrav1.NewCertificateServiceServer(certificate.New(certificate.Config{
		Logger:        logger,
		DB:            database,
		Vault:         acmeVaultSvc,
		EmailDomain:   cfg.Acme.EmailDomain,
		DefaultDomain: cfg.DefaultDomain,
		DNSProvider:   dnsProvider,
		HTTPProvider:  httpProvider,
	}), restate.WithInactivityTimeout(15*time.Minute)))

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
				req, err := http.NewRequestWithContext(ctx, http.MethodPost, registerURL, bytes.NewBufferString(payload))
				if err != nil {
					return fmt.Errorf("failed to create registration request: %w", err)
				}

				req.Header.Set("Content-Type", "application/json")

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return fmt.Errorf("failed to register with Restate: %w", err)
				}
				defer func() { _ = resp.Body.Close() }()

				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					return nil
				}

				return fmt.Errorf("registration returned status %d", resp.StatusCode)
			})

			if err != nil {
				logger.Error("failed to register with Restate after retries", "error", err.Error())
			} else {
				logger.Info("Successfully registered with Restate")

				// Bootstrap wildcard certificate for default domain if ACME is enabled
				if cfg.Acme.Enabled && dnsProvider != nil && cfg.DefaultDomain != "" {
					bootstrapWildcardDomain(ctx, database, logger, cfg.DefaultDomain)
				}

				// Start the certificate renewal cron job if ACME is enabled
				// Use Send with idempotency key so multiple restarts don't create duplicate crons
				if cfg.Acme.Enabled && dnsProvider != nil {
					certClient := hydrav1.NewCertificateServiceIngressClient(restateClient, "global")
					_, startErr := certClient.RenewExpiringCertificates().Send(
						ctx,
						&hydrav1.RenewExpiringCertificatesRequest{
							DaysBeforeExpiry: 30,
						},
						restate.WithIdempotencyKey("cert-renewal-cron-startup"),
					)
					if startErr != nil {
						logger.Warn("failed to start certificate renewal cron", "error", startErr)
					} else {
						logger.Info("Certificate renewal cron job started")
					}
				}
			}
		}()
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
		DomainCache:    caches.Domains,
		ChallengeCache: caches.Challenges,
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

// bootstrapWildcardDomain ensures a wildcard domain and ACME challenge exist for the default domain.
//
// This helper function creates the necessary database records for automatic
// wildcard certificate issuance during startup. It checks if the wildcard
// domain already exists and creates both the custom domain record and
// ACME challenge record if needed.
//
// The function uses "unkey_internal" as the workspace ID for
// platform-managed resources, ensuring separation from user workspaces.
//
// This is called during control plane startup when ACME is enabled and
// a default domain is configured, allowing the renewal cron job to
// automatically issue wildcard certificates without manual intervention.
func bootstrapWildcardDomain(ctx context.Context, database db.Database, logger logging.Logger, defaultDomain string) {
	wildcardDomain := "*." + defaultDomain

	// Check if the wildcard domain already exists
	_, err := db.Query.FindCustomDomainByDomain(ctx, database.RO(), wildcardDomain)
	if err == nil {
		logger.Info("Wildcard domain already exists", "domain", wildcardDomain)
		return
	}
	if !db.IsNotFound(err) {
		logger.Error("Failed to check for existing wildcard domain", "error", err, "domain", wildcardDomain)
		return
	}

	// Create the custom domain record
	domainID := uid.New(uid.DomainPrefix)
	now := time.Now().UnixMilli()

	// Use "unkey_internal" as the workspace for platform-managed resources
	workspaceID := "unkey_internal"
	err = db.Query.UpsertCustomDomain(ctx, database.RW(), db.UpsertCustomDomainParams{
		ID:            domainID,
		WorkspaceID:   workspaceID,
		Domain:        wildcardDomain,
		ChallengeType: db.CustomDomainsChallengeTypeDNS01,
		CreatedAt:     now,
		UpdatedAt:     sql.NullInt64{Int64: now, Valid: true},
	})
	if err != nil {
		logger.Error("Failed to create wildcard domain", "error", err, "domain", wildcardDomain)
		return
	}

	// Create the ACME challenge record with status 'waiting' so the renewal cron picks it up
	err = db.Query.InsertAcmeChallenge(ctx, database.RW(), db.InsertAcmeChallengeParams{
		WorkspaceID:   workspaceID,
		DomainID:      domainID,
		Token:         "",
		Authorization: "",
		Status:        db.AcmeChallengesStatusWaiting,
		ChallengeType: db.AcmeChallengesChallengeTypeDNS01,
		CreatedAt:     now,
		UpdatedAt:     sql.NullInt64{Int64: now, Valid: true},
		ExpiresAt:     0, // Will be set when certificate is issued
	})
	if err != nil {
		logger.Error("Failed to create ACME challenge for wildcard domain", "error", err, "domain", wildcardDomain)
		return
	}

	logger.Info("Bootstrapped wildcard domain for certificate issuance", "domain", wildcardDomain)
}
