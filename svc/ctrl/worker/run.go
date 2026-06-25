package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/go-acme/lego/v4/challenge"
	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	restate "github.com/restatedev/sdk-go"
	restateServer "github.com/restatedev/sdk-go/server"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/buildinfo"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/otel"
	"github.com/unkeyed/unkey/pkg/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
	"github.com/unkeyed/unkey/pkg/ptr"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/pkg/rpc/interceptor"
	"github.com/unkeyed/unkey/pkg/runner"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/services/acme/providers"
	workerapp "github.com/unkeyed/unkey/svc/ctrl/worker/app"
	"github.com/unkeyed/unkey/svc/ctrl/worker/buildslot"
	"github.com/unkeyed/unkey/svc/ctrl/worker/certificate"
	"github.com/unkeyed/unkey/svc/ctrl/worker/clickhouseuser"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deploybilling"
	workercustomdomain "github.com/unkeyed/unkey/svc/ctrl/worker/customdomain"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deploy"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deployment"
	workerenvironment "github.com/unkeyed/unkey/svc/ctrl/worker/environment"
	githubclient "github.com/unkeyed/unkey/svc/ctrl/worker/github"
	"github.com/unkeyed/unkey/svc/ctrl/worker/githubstatus"
	"github.com/unkeyed/unkey/svc/ctrl/worker/githubwebhook"
	"github.com/unkeyed/unkey/svc/ctrl/worker/keylastusedsync"

	ratelimitdb "github.com/unkeyed/unkey/internal/services/ratelimit/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/openapi"
	workerproject "github.com/unkeyed/unkey/svc/ctrl/worker/project"
	"github.com/unkeyed/unkey/svc/ctrl/worker/routing"
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
	if cfg.InstanceID == "" {
		cfg.InstanceID = uid.New(uid.InstancePrefix)
	}
	if cfg.Clock == nil {
		cfg.Clock = clock.New()
	}
	cfg.CnameDomain = strings.TrimSuffix(strings.TrimSpace(cfg.CnameDomain), ".")

	// Disable CNAME following in lego to prevent it from following wildcard CNAMEs
	// (e.g., *.example.com -> loadbalancer.aws.com) and failing Route53 zone lookup.
	// Must be set before creating any ACME DNS providers.
	_ = os.Setenv("LEGO_DISABLE_CNAME_SUPPORT", "true")

	// Initialize OTEL before logger so logger picks up OTLP handler
	var shutdownGrafana func(context.Context) error
	var err error
	if cfg.Observability.Tracing != nil {
		shutdownGrafana, err = otel.InitGrafana(ctx, otel.Config{
			Application:        "worker",
			InstanceID:         cfg.InstanceID,
			CloudRegion:        cfg.Region,
			TraceSampleRate:    cfg.Observability.Tracing.SampleRate,
			PrometheusGatherer: nil,
		})
		if err != nil {
			return fmt.Errorf("unable to init grafana: %w", err)
		}
	}

	// Add base attributes to global logger
	logger.AddBaseAttrs(slog.GroupAttrs("instance",
		slog.String("id", cfg.InstanceID),
		slog.String("region", cfg.Region),
		slog.String("version", buildinfo.Version),
	))

	r := runner.New()
	defer r.Recover()

	r.DeferCtx(shutdownGrafana)

	reg := promclient.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	//nolint:exhaustruct
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	reg.MustRegister(prometheus.NewSystemMetricsCollector())
	lazy.SetRegistry(reg)
	buildinfo.RegisterBuildInfoMetrics("worker")

	// Create vault client for remote vault service
	var vaultClient vault.VaultServiceClient
	if cfg.Vault.URL != "" {
		vaultClient = vault.NewConnectVaultServiceClient(vaultv1connect.NewVaultServiceClient(
			http.DefaultClient,
			cfg.Vault.URL,
			connect.WithInterceptors(interceptor.NewHeaderInjector(map[string]string{
				"Authorization": "Bearer " + cfg.Vault.Token,
			})),
		))
		logger.Info("Vault client initialized", "url", cfg.Vault.URL)
	}

	// Initialize database
	database, err := db.New(cfg.Database.Primary)
	if err != nil {
		return fmt.Errorf("unable to create db: %w", err)
	}

	r.Defer(database.Close)

	// Create GitHub client for deploy workflow (optional)
	var ghClient githubclient.GitHubClient = githubclient.NewNoop()
	if cfg.GitHub != nil {
		client, ghErr := githubclient.NewClient(githubclient.ClientConfig{
			AppID:         cfg.GitHub.AppID,
			PrivateKeyPEM: cfg.GitHub.PrivateKeyPEM,
			WebhookSecret: "",
		})
		if ghErr != nil {
			return fmt.Errorf("failed to create GitHub client: %w", ghErr)
		}
		ghClient = client
		logger.Info("GitHub client initialized")
	} else {
		logger.Info("GitHub client disabled (credentials not configured)")
	}

	var ch clickhouse.ClickHouse = clickhouse.NewNoop()
	// Billing usage reader is the concrete *clickhouse.Client: the meter query
	// (GetInstanceMeterUsage) is not on the ClickHouse interface. Nil until
	// ClickHouse is configured, which leaves the billing push disabled.
	var billingUsageReader deploybilling.UsageReader
	buildSteps := batch.NewNoop[schema.BuildStepV1]()
	buildStepLogs := batch.NewNoop[schema.BuildStepLogV1]()

	if cfg.ClickHouse.URL != "" {
		chClient, chErr := clickhouse.New(clickhouse.Config{
			URL: cfg.ClickHouse.URL,
		})
		if chErr != nil {
			logger.Error("failed to create clickhouse client, continuing with noop", "error", chErr)
		} else {
			ch = chClient
			billingUsageReader = chClient

			buildSteps = clickhouse.NewBuffer[schema.BuildStepV1](chClient, "default.build_steps_v1", clickhouse.BufferConfig{
				Name:          "build_steps",
				BatchSize:     1_000,
				BufferSize:    2_000,
				FlushInterval: 2 * time.Second,
				Consumers:     1,
				Drop:          true,
				OnFlushError:  nil,
			})
			buildStepLogs = clickhouse.NewBuffer[schema.BuildStepLogV1](chClient, "default.build_step_logs_v1", clickhouse.BufferConfig{
				Name:          "build_step_logs",
				BatchSize:     1_000,
				BufferSize:    2_000,
				FlushInterval: 2 * time.Second,
				Consumers:     1,
				Drop:          true,
				OnFlushError:  nil,
			})

			// Close connection last (LIFO: first registered closes last)
			r.Defer(chClient.Close)
			r.Defer(func() error { buildSteps.Close(); return nil })
			r.Defer(func() error { buildStepLogs.Close(); return nil })
		}
	}

	// Restate Server. The SDK logs "Handling invocation" / "Invocation
	// completed successfully" at INFO on every invocation, and several crons
	// tick every minute, so its INFO chatter drowns out real signal. Give it a
	// WARN-gated view of our handler: its warnings and errors (e.g. invocation
	// panics) still surface, its routine success noise is dropped, and the
	// app's own INFO/DEBUG logs (which honor UNKEY_LOG_LEVEL) are unaffected.
	restateSrv := restateServer.NewRestate().WithLogger(logger.AtLevel(logger.GetHandler(), slog.LevelWarn), false)

	// Shared Restate admin client used both for service registration and
	// for in-flight invocation cancellation (e.g. superseded sibling cleanup).
	restateAdminClient := restateadmin.New(restateadmin.Config{
		BaseURL: cfg.Restate.AdminURL,
		APIKey:  cfg.Restate.APIKey,
	})

	buildPlatform, err := cfg.GetBuildPlatform()
	if err != nil {
		return fmt.Errorf("invalid build platform: %w", err)
	}

	restateSrv.Bind(hydrav1.NewDeployServiceServer(deploy.New(deploy.Config{
		DB:            database,
		DefaultDomain: cfg.DefaultDomain,
		Vault:         vaultClient,

		GitHub:                          ghClient,
		RegistryConfig:                  deploy.RegistryConfig(cfg.GetRegistryConfig()),
		BuildPlatform:                   deploy.BuildPlatform(buildPlatform),
		DepotConfig:                     deploy.DepotConfig(cfg.GetDepotConfig()),
		Clickhouse:                      ch,
		BuildSteps:                      buildSteps,
		BuildStepLogs:                   buildStepLogs,
		AllowUnauthenticatedDeployments: ptr.SafeDeref(cfg.GitHub).AllowUnauthenticatedDeployments,
		DashboardURL:                    cfg.DashboardURL,
	}),
		// Retry with exponential backoff: 2s → 4s → 8s → 16s → 30s (capped),
		// 15 attempts (~5 min total). Short backoffs keep the worst-case
		// cancel latency low — a user-initiated cancel only lands at the
		// next attempt boundary, so longer intervals make cancels feel
		// stuck. 5 minutes total is enough for transient blips; persistent
		// failures should surface fast rather than retry for half an hour.
		//
		// PauseOnMaxAttempts (not Kill) so compensations can still run:
		// on KILL the invocation is torn down without re-entering the
		// handler, so the Go defer that fires compensation.Execute never
		// runs. Individual restate.Run calls should each set
		// WithMaxRetryDuration so they return TerminalError into Go on
		// exhaustion — that's the normal path. This service-level policy
		// is a safety net for failures that escape Run-level bounds.
		restate.WithInvocationRetryPolicy(
			restate.WithInitialInterval(2*time.Second),
			restate.WithExponentiationFactor(2.0),
			restate.WithMaxInterval(30*time.Second),
			restate.WithMaxAttempts(15),
			restate.PauseOnMaxAttempts(),
		),
	))
	restateSrv.Bind(hydrav1.NewDeploymentServiceServer(deployment.New(deployment.Config{
		DB: database,
	}), restate.WithIngressPrivate(true)))

	restateSrv.Bind(hydrav1.NewGitHubStatusServiceServer(githubstatus.New(githubstatus.Config{
		GitHub: ghClient,
		DB:     database,
	}), restate.WithIngressPrivate(true)))

	restateSrv.Bind(hydrav1.NewRoutingServiceServer(routing.New(routing.Config{
		DB:            database,
		DefaultDomain: cfg.DefaultDomain,
	}), restate.WithIngressPrivate(true)))

	restateSrv.Bind(hydrav1.NewOpenapiServiceServer(openapi.New(openapi.Config{
		DB: database,
	}), restate.WithIngressPrivate(true),
		// Retry with exponential backoff: 1m → 2m → 4m → 8m → 10m (capped), ~1 hour total.
		// Scraping is best-effort (fire-and-forget from deploy); bound retries to avoid
		// wasting resources on permanently broken endpoints. Pause (not kill) on
		// exhaustion so any future compensation logic can run via re-entry.
		restate.WithInvocationRetryPolicy(
			restate.WithInitialInterval(1*time.Minute),
			restate.WithExponentiationFactor(2.0),
			restate.WithMaxInterval(10*time.Minute),
			restate.WithMaxAttempts(10),
			restate.PauseOnMaxAttempts(),
		),
	))

	restateSrv.Bind(hydrav1.NewGitHubWebhookServiceServer(githubwebhook.New(githubwebhook.Config{
		DB:                              database,
		GitHub:                          ghClient,
		RestateAdmin:                    restateAdminClient,
		DashboardURL:                    cfg.DashboardURL,
		AllowUnauthenticatedDeployments: ptr.SafeDeref(cfg.GitHub).AllowUnauthenticatedDeployments,
	})))

	restateSrv.Bind(hydrav1.NewProjectServiceServer(workerproject.New(workerproject.Config{
		DB: database,
	})))
	restateSrv.Bind(hydrav1.NewAppServiceServer(workerapp.New(workerapp.Config{
		DB: database,
	})))
	envSvc, err := workerenvironment.New(workerenvironment.Config{
		DB:    database,
		Admin: restateAdminClient,
	})
	if err != nil {
		return fmt.Errorf("failed to create environment worker service: %w", err)
	}
	restateSrv.Bind(hydrav1.NewEnvironmentServiceServer(envSvc))

	// BuildSlotService is short-lived coordination — AcquireOrWait/Release
	// journals have no debugging value (each invocation just reads state,
	// maybe resolves an awakeable, and returns), so keep their retention
	// minimal.
	restateSrv.Bind(hydrav1.NewBuildSlotServiceServer(buildslot.New(buildslot.Config{
		DB: database,
	}),
		restate.WithIngressPrivate(true),
		restate.WithJournalRetention(1*time.Minute),
	))

	restateSrv.Bind(hydrav1.NewCustomDomainServiceServer(workercustomdomain.New(workercustomdomain.Config{
		DB:          database,
		CnameDomain: cfg.CnameDomain,
	}),
		// Retry every 1 minute for up to 24 hours (1440 attempts). Pause (not
		// kill) on exhaustion so compensations remain possible via operator
		// cancel.
		restate.WithInvocationRetryPolicy(
			restate.WithInitialInterval(1*time.Minute),
			restate.WithExponentiationFactor(1.0), // Fixed interval, no exponential backoff
			restate.WithMaxInterval(1*time.Minute),
			restate.WithMaxAttempts(1440),
			restate.PauseOnMaxAttempts(),
		),
	))

	// Initialize domain cache for ACME providers
	clk := cfg.Clock
	if clk == nil {
		clk = clock.New()
	}
	domainCache, domainCacheErr := cache.New(cache.Config[string, db.CustomDomain]{
		Fresh:    5 * time.Minute,
		Stale:    10 * time.Minute,
		MaxSize:  10000,
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
	var certHeartbeat healthcheck.Heartbeat = healthcheck.NewNoop()
	if cfg.Heartbeat.CertRenewalURL != "" {
		certHeartbeat = healthcheck.NewChecklyHeartbeat(cfg.Heartbeat.CertRenewalURL)
	}
	restateSrv.Bind(hydrav1.NewCertificateServiceServer(certificate.New(certificate.Config{
		DB:            database,
		Vault:         vaultClient,
		EmailDomain:   cfg.Acme.EmailDomain,
		DefaultDomain: cfg.DefaultDomain,
		DNSProvider:   dnsProvider,
		HTTPProvider:  httpProvider,
		Heartbeat:     certHeartbeat,
	}), restate.WithInactivityTimeout(15*time.Minute)))

	// ClickHouse user provisioning service (optional - requires admin URL and vault)
	if cfg.ClickHouse.AdminURL == "" {
		logger.Info("ClickhouseUserService disabled: clickhouse admin_url not configured")
	} else if vaultClient == nil {
		logger.Warn("ClickhouseUserService disabled: vault not configured")
	} else {
		chAdmin, chAdminErr := clickhouse.New(clickhouse.Config{
			URL: cfg.ClickHouse.AdminURL,
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
			})))
			logger.Info("ClickhouseUserService enabled")
		}
	}

	// Unified CronService — every scheduled task is a handler on one
	// hydra.v1.CronService VO. No service-level retry/journal-retention
	// defaults: each task previously lived in its own service with its
	// own (or no) overrides, and a blanket default would silently change
	// failure semantics — e.g. forcing PauseOnMaxAttempts on singleton-
	// keyed VOs wedges every subsequent tick under that key. Per-handler
	// options below mirror each task's pre-consolidation behavior.
	cronSvc, err := cron.New(cron.Config{
		DB:                        database,
		Clickhouse:                ch,
		Clock:                     clk,
		RatelimitDB:               ratelimitdb.New(database.RW(), database.RO()),
		SlackQuotaCheckWebhookURL: cfg.Slack.QuotaCheckWebhookURL,
		BillingUsageReader:        billingUsageReader,
		StripeSecretKey:           cfg.Billing.StripeSecretKey,
		Heartbeats: cron.Heartbeats{
			QuotaCheck:        cronHeartbeat(cfg.Heartbeat.QuotaCheckURL),
			KeyRefill:         cronHeartbeat(cfg.Heartbeat.KeyRefillURL),
			KeyLastUsedSync:   cronHeartbeat(cfg.Heartbeat.KeyLastUsedSyncURL),
			AuditLogExport:    cronHeartbeat(cfg.Heartbeat.AuditLogExportURL),
			AuditLogCleanup:   cronHeartbeat(cfg.Heartbeat.AuditLogOutboxCleanupURL),
			DeployBillingPush: cronHeartbeat(cfg.Heartbeat.DeployBillingPushURL),
		},
	})
	if err != nil {
		return fmt.Errorf("create cron service: %w", err)
	}
	// Tighter policy for KeyLastUsedSync: deadlocks resolve fast and
	// the orchestrator just fans out to partition VOs, so capping retries
	// short makes a wedged sync visible quickly. Kill (not Pause) on
	// exhaustion: the orchestrator fans out to 8 partition children and
	// waits on each future sequentially, so a paused partition blocks
	// the whole sync until an operator cancels it (incident: partitions
	// 0/1/6/7 hit a Restate 404 and sat paused, suspending the run). The
	// sync is idempotent and cron-triggered, has no compensation, and
	// surfaces failures via the missing end-of-run heartbeat — so killing
	// on retry exhaustion lets the orchestrator's future resolve with a
	// terminal error, the run fails fast, and the next cron tick retries
	// from the persisted cursor.
	cronKeyLastUsedRetry := restate.WithInvocationRetryPolicy(
		restate.WithInitialInterval(100*time.Millisecond),
		restate.WithExponentiationFactor(2.0),
		restate.WithMaxInterval(5*time.Second),
		restate.WithMaxAttempts(5),
		restate.KillOnMaxAttempts(),
	)
	// Ratelimit global-counters cleanup keeps the pre-consolidation
	// 5-attempt / 100ms-5s policy with PauseOnMaxAttempts: stateless
	// DELETE that the hot path doesn't depend on, so pausing for an
	// operator to inspect a real failure is better than killing silently.
	cronRatelimitGCCRetry := restate.WithInvocationRetryPolicy(
		restate.WithInitialInterval(100*time.Millisecond),
		restate.WithExponentiationFactor(2.0),
		restate.WithMaxInterval(5*time.Second),
		restate.WithMaxAttempts(5),
		restate.PauseOnMaxAttempts(),
	)
	// AuditLogOutboxCleanup mirrors the ratelimit cleanup policy: a
	// stateless, cutoff-bounded DELETE the hot path doesn't depend on, so
	// pausing for an operator to inspect a real failure beats killing
	// silently. The daily cadence means a paused run is caught well before
	// the next tick.
	cronAuditLogCleanupRetry := restate.WithInvocationRetryPolicy(
		restate.WithInitialInterval(100*time.Millisecond),
		restate.WithExponentiationFactor(2.0),
		restate.WithMaxInterval(5*time.Second),
		restate.WithMaxAttempts(5),
		restate.PauseOnMaxAttempts(),
	)
	// AuditLogExport runs every minute and is idempotent: any failure is
	// recovered by the next tick, not by replaying journals from
	// yesterday. 1h journal retention keeps enough debugging headroom for
	// an oncall to inspect a recent failure without bloating the journal
	// store with ~1440 dead invocations/day. No retry override — SDK
	// default behavior was the pre-consolidation contract.
	restateSrv.Bind(hydrav1.NewCronServiceServer(cronSvc).
		ConfigureHandler("RunKeyLastUsedSync", cronKeyLastUsedRetry).
		ConfigureHandler("RunRatelimitGlobalCountersCleanup", cronRatelimitGCCRetry).
		ConfigureHandler("RunAuditLogOutboxCleanup", cronAuditLogCleanupRetry).
		ConfigureHandler("RunAuditLogExport", restate.WithJournalRetention(1*time.Hour)))
	logger.Info("CronService enabled")

	// KeyLastUsedPartitionService is the per-partition VO fanned out from
	// the orchestrator. Stays standalone — it's not cron-triggered.
	keyLastUsedPartitionSvc, err := keylastusedsync.NewPartitionService(keylastusedsync.PartitionConfig{
		DB:         database,
		Clickhouse: ch,
	})
	if err != nil {
		return fmt.Errorf("create key last used partition service: %w", err)
	}
	restateSrv.Bind(hydrav1.NewKeyLastUsedPartitionServiceServer(keyLastUsedPartitionSvc, cronKeyLastUsedRetry))
	logger.Info("KeyLastUsedPartitionService enabled")

	// Get the Restate handler and mount it on a mux with health endpoint
	restateHandler, err := restateSrv.Handler()
	if err != nil {
		return fmt.Errorf("failed to get restate handler: %w", err)
	}

	mux := http.NewServeMux()
	r.RegisterHealth(mux)
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

	r.DeferCtx(server.Shutdown)
	r.Go(func(ctx context.Context) error {
		logger.Info("Starting worker server", "addr", addr)
		if serveErr := server.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
			return fmt.Errorf("server failed: %w", serveErr)
		}
		return nil
	})

	// Register with Restate admin API (only if RegisterAs is configured)
	// In k8s environments, registration is handled externally
	if cfg.Restate.RegisterAs != "" {
		r.Go(func(ctx context.Context) error {
			logger.Info("Registering with Restate", "service_uri", cfg.Restate.RegisterAs)
			if err := restateAdminClient.RegisterDeployment(ctx, cfg.Restate.RegisterAs, true); err != nil {
				logger.Error("failed to register with Restate", "error", err)
				return err
			}
			logger.Info("Successfully registered with Restate")
			return nil
		})
	} else {
		logger.Info("Skipping Restate registration (restate-register-as not configured)")
	}

	if cfg.Observability.Metrics != nil && cfg.Observability.Metrics.PrometheusPort > 0 {
		prom, promErr := prometheus.NewWithRegistry(reg)
		if promErr != nil {
			return fmt.Errorf("failed to create prometheus server: %w", promErr)
		}

		ln, lnErr := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Observability.Metrics.PrometheusPort))
		if lnErr != nil {
			return fmt.Errorf("unable to listen on port %d: %w", cfg.Observability.Metrics.PrometheusPort, lnErr)
		}

		r.DeferCtx(prom.Shutdown)
		r.Go(func(ctx context.Context) error {
			logger.Info("prometheus started", "port", cfg.Observability.Metrics.PrometheusPort)
			if serveErr := prom.Serve(ctx, ln); serveErr != nil && !errors.Is(serveErr, context.Canceled) {
				return fmt.Errorf("failed to start prometheus server: %w", serveErr)
			}
			return nil
		})
	}

	// Wait for signal and handle shutdown
	logger.Info("Worker started successfully")
	if err := r.Wait(ctx); err != nil {
		logger.Error("Shutdown failed", "error", err)
		return err
	}

	logger.Info("Worker shut down successfully")
	return nil
}

// cronHeartbeat returns a Checkly heartbeat for url, or a noop if url is
// empty. Used to wire each cron task's monitoring URL without scattering
// nil-or-noop branches through the cron service.
func cronHeartbeat(url string) healthcheck.Heartbeat {
	if url == "" {
		return healthcheck.NewNoop()
	}
	return healthcheck.NewChecklyHeartbeat(url)
}
