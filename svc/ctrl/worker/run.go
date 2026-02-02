package worker

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	"github.com/go-acme/lego/v4/challenge"
	restate "github.com/restatedev/sdk-go"
	restateServer "github.com/restatedev/sdk-go/server"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/prometheus"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/pkg/rpc/interceptor"
	"github.com/unkeyed/unkey/pkg/shutdown"
	"github.com/unkeyed/unkey/svc/ctrl/pkg/build"
	"github.com/unkeyed/unkey/svc/ctrl/pkg/s3"
	"github.com/unkeyed/unkey/svc/ctrl/services/acme/providers"
	"github.com/unkeyed/unkey/svc/ctrl/worker/certificate"
	"github.com/unkeyed/unkey/svc/ctrl/worker/clickhouseuser"
	workercustomdomain "github.com/unkeyed/unkey/svc/ctrl/worker/customdomain"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deploy"
	"github.com/unkeyed/unkey/svc/ctrl/worker/routing"
	"github.com/unkeyed/unkey/svc/ctrl/worker/versioning"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// Run starts the Restate worker service with the provided configuration.
//
// This function initializes all required services and starts the Restate server
// for workflow execution. It performs these major initialization steps:
//  1. Validates configuration and initializes structured logging
//  2. Creates vault services for secrets and ACME certificates
//  3. Initializes database and build storage
//  4. Creates build service (docker/depot backend)
//  5. Initializes ACME caches and providers (HTTP-01, DNS-01)
//  6. Starts Restate server with workflow service bindings
//  7. Registers with Restate admin API for service discovery
//  8. Starts health check endpoint
//  9. Optionally starts Prometheus metrics server
//
// The worker handles graceful shutdown when context is cancelled, properly
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

	logger := logging.New()
	if cfg.InstanceID != "" {
		logger = logger.With(slog.String("instanceID", cfg.InstanceID))
	}

	// Create vault client for remote vault service
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

	imageStore, err := s3.NewS3(s3.S3Config{
		Logger:            logger,
		S3URL:             cfg.BuildS3.URL,
		S3PresignURL:      cfg.BuildS3.ExternalURL,
		S3Bucket:          cfg.BuildS3.Bucket,
		S3AccessKeyID:     cfg.BuildS3.AccessKeyID,
		S3AccessKeySecret: cfg.BuildS3.AccessKeySecret,
	})
	if err != nil {
		return fmt.Errorf("unable to create build storage: %w", err)
	}

	var ch clickhouse.ClickHouse = clickhouse.NewNoop()
	if cfg.ClickhouseURL != "" {
		chClient, chErr := clickhouse.New(clickhouse.Config{
			URL:    cfg.ClickhouseURL,
			Logger: logger,
		})
		if chErr != nil {
			logger.Error("failed to create clickhouse client, continuing with noop", "error", chErr)
		} else {
			ch = chClient
		}
	}

	// Restate Server
	restateSrv := restateServer.NewRestate().WithLogger(logging.Handler(), false)

	restateSrv.Bind(hydrav1.NewBuildServiceServer(build.New(build.Config{
		InstanceID:     cfg.InstanceID,
		DB:             database,
		RegistryConfig: build.RegistryConfig(cfg.GetRegistryConfig()),
		BuildPlatform:  build.BuildPlatform(cfg.GetBuildPlatform()),
		DepotConfig:    build.DepotConfig(cfg.GetDepotConfig()),
		Clickhouse:     ch,
		Logger:         logger,
	})))

	restateSrv.Bind(hydrav1.NewDeploymentServiceServer(deploy.New(deploy.Config{
		Logger:           logger,
		DB:               database,
		DefaultDomain:    cfg.DefaultDomain,
		Vault:            vaultClient,
		SentinelImage:    cfg.SentinelImage,
		AvailableRegions: cfg.AvailableRegions,
		BuildStorage:     imageStore,
	})))

	restateSrv.Bind(hydrav1.NewRoutingServiceServer(routing.New(routing.Config{
		Logger:        logger,
		DB:            database,
		DefaultDomain: cfg.DefaultDomain,
	}), restate.WithIngressPrivate(true)))

	restateSrv.Bind(hydrav1.NewVersioningServiceServer(versioning.New(), restate.WithIngressPrivate(true)))

	restateSrv.Bind(hydrav1.NewCustomDomainServiceServer(workercustomdomain.New(workercustomdomain.Config{
		DB:          database,
		Logger:      logger,
		CnameDomain: cfg.CnameDomain,
	}),
		// Retry every 1 minute for up to 24 hours (1440 attempts)
		restate.WithInvocationRetryPolicy(
			restate.WithInitialInterval(1*time.Minute),
			restate.WithExponentiationFactor(1.0), // Fixed interval, no exponential backoff
			restate.WithMaxInterval(1*time.Minute),
			restate.WithMaxAttempts(1440),
			restate.KillOnMaxAttempts(),
		),
	))

	// Initialize domain cache for ACME providers
	clk := clock.New()
	domainCache, domainCacheErr := cache.New(cache.Config[string, db.CustomDomain]{
		Fresh:    5 * time.Minute,
		Stale:    10 * time.Minute,
		MaxSize:  10000,
		Logger:   logger,
		Resource: "domains",
		Clock:    clk,
	})
	if domainCacheErr != nil {
		return fmt.Errorf("failed to create domain cache: %w", domainCacheErr)
	}

	// Setup ACME challenge providers
	var dnsProvider challenge.Provider
	var httpProvider challenge.Provider
	if cfg.Acme.Enabled {
		// HTTP-01 provider for regular (non-wildcard) domains
		httpProv, httpErr := providers.NewHTTPProvider(providers.HTTPConfig{
			DB:          database,
			Logger:      logger,
			DomainCache: domainCache,
		})
		if httpErr != nil {
			return fmt.Errorf("failed to create HTTP-01 provider: %w", httpErr)
		}
		httpProvider = httpProv
		logger.Info("ACME HTTP-01 provider enabled")

		// DNS-01 provider for wildcard domains (requires DNS provider config)
		if cfg.Acme.Route53.Enabled {
			r53Provider, r53Err := providers.NewRoute53Provider(providers.Route53Config{
				DB:              database,
				Logger:          logger,
				AccessKeyID:     cfg.Acme.Route53.AccessKeyID,
				SecretAccessKey: cfg.Acme.Route53.SecretAccessKey,
				Region:          cfg.Acme.Route53.Region,
				HostedZoneID:    cfg.Acme.Route53.HostedZoneID,
				DomainCache:     domainCache,
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
		Vault:         vaultClient,
		EmailDomain:   cfg.Acme.EmailDomain,
		DefaultDomain: cfg.DefaultDomain,
		DNSProvider:   dnsProvider,
		HTTPProvider:  httpProvider,
	}), restate.WithInactivityTimeout(15*time.Minute)))

	// ClickHouse user provisioning service (optional - requires admin URL and vault)
	if cfg.ClickhouseAdminURL == "" {
		logger.Info("ClickhouseUserService disabled: CLICKHOUSE_ADMIN_URL not configured")
	} else if vaultClient == nil {
		logger.Warn("ClickhouseUserService disabled: vault not configured")
	} else {
		chAdmin, chAdminErr := clickhouse.New(clickhouse.Config{
			URL:    cfg.ClickhouseAdminURL,
			Logger: logger,
		})
		if chAdminErr != nil {
			logger.Warn("ClickhouseUserService disabled: failed to connect to admin",
				"error", chAdminErr,
			)
		} else {
			restateSrv.Bind(hydrav1.NewClickhouseUserServiceServer(clickhouseuser.New(clickhouseuser.Config{
				DB:         database,
				Vault:      vaultClient,
				Clickhouse: chAdmin,
				Logger:     logger,
			})))
			logger.Info("ClickhouseUserService enabled")
		}
	}

	// Get the Restate handler and mount it on a mux with health endpoint
	restateHandler, err := restateSrv.Handler()
	if err != nil {
		return fmt.Errorf("failed to get restate handler: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("/", restateHandler)

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
	addr := fmt.Sprintf(":%d", cfg.Restate.HttpPort)
	server := &http.Server{
		Addr:              addr,
		Handler:           h2cHandler,
		ReadHeaderTimeout: 30 * time.Second,
	}

	go func() {
		logger.Info("Starting worker server", "addr", addr)
		if serveErr := server.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
			logger.Error("server failed", "error", serveErr)
		}
	}()

	shutdowns.RegisterCtx(server.Shutdown)

	// Register with Restate admin API (only if RegisterAs is configured)
	// In k8s environments, registration is handled externally
	if cfg.Restate.RegisterAs != "" {
		adminClient := restateadmin.New(restateadmin.Config{
			BaseURL: cfg.Restate.AdminURL,
			APIKey:  cfg.Restate.APIKey,
		})
		go func() {
			logger.Info("Registering with Restate", "service_uri", cfg.Restate.RegisterAs)
			if err := adminClient.RegisterDeployment(ctx, cfg.Restate.RegisterAs); err != nil {
				logger.Error("failed to register with Restate", "error", err)
				return
			}
			logger.Info("Successfully registered with Restate")
		}()
	} else {
		logger.Info("Skipping Restate registration (restate-register-as not configured)")
	}

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
	logger.Info("Worker started successfully")
	if err := shutdowns.WaitForSignal(ctx); err != nil {
		logger.Error("Shutdown failed", "error", err)
		return err
	}

	logger.Info("Worker shut down successfully")
	return nil
}
