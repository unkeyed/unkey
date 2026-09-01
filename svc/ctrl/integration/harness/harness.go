// Package harness provides a unified test harness for ctrl worker integration tests.
// It starts all required services (MySQL, ClickHouse, Restate, Vault) and registers
// all worker handlers, enabling end-to-end testing of any handler without per-test setup.
package harness

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	ratelimitdb "github.com/unkeyed/unkey/internal/services/ratelimit/db"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/billingmeter"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/invoicecloser"
	"github.com/unkeyed/unkey/svc/ctrl/worker/buildslot"
	"github.com/unkeyed/unkey/svc/ctrl/worker/clickhouseuser"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deploybilling"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deploy"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deployteardown"
	"github.com/unkeyed/unkey/svc/ctrl/worker/keylastusedsync"
	vaulttestutil "github.com/unkeyed/unkey/svc/vault/testutil"
)

// Harness holds all test dependencies for ctrl worker integration tests.
// It provides access to databases, clients, and the Restate ingress URL.
type Harness struct {
	// Ctx is the test context with timeout.
	Ctx context.Context

	// DB is the MySQL database connection.
	DB db.Database

	// Seed provides methods to create test entities in MySQL.
	Seed *seed.Seeder

	// ClickHouseSeed provides methods to insert test data in ClickHouse.
	ClickHouseSeed *seed.ClickHouseSeeder

	// ClickHouse is the ClickHouse client for analytics queries.
	ClickHouse clickhouse.ClickHouse

	// ClickHouseConn is a direct ClickHouse connection for inserting test data.
	ClickHouseConn ch.Conn

	// ClickHouseDSN is the ClickHouse connection string.
	ClickHouseDSN string

	// VaultClient is a real vault client for encryption/decryption.
	VaultClient vault.VaultServiceClient

	// VaultToken is the bearer token for the vault service.
	VaultToken string

	// Restate is the ingress client for calling Restate services.
	Restate *ingress.Client

	// RestateAdmin is the URL for Restate admin operations.
	RestateAdmin string

	// Clock is the clock instance wired into the cron service. Defaults
	// to clock.New() (real time); tests that need to drive cutoffs can
	// pass a *clock.TestClock via WithClock and assert against it here.
	Clock clock.Clock
}

// Option configures the test harness.
type Option func(*harnessOpts)

type harnessOpts struct {
	timeout time.Duration
	clock   clock.Clock

	billingUsageReader deploybilling.UsageReader
	billingPusher      billingmeter.Pusher
	billingCloser      invoicecloser.Closer
}

// WithTimeout overrides the default harness context timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(o *harnessOpts) {
		o.timeout = timeout
	}
}

// WithClock injects a clock into the cron service. Use clock.NewTestClock()
// to drive cutoff timestamps deterministically (e.g. for the ratelimit
// global-counters cleanup handler, which reads s.clock.Now()).
func WithClock(c clock.Clock) Option {
	return func(o *harnessOpts) {
		o.clock = c
	}
}

// WithDeployBilling injects the Deploy billing dependencies into the cron
// service so tests can drive RunDeployBillingPush / RunDeployBillingClose
// against fakes instead of real ClickHouse usage and Stripe. The reader feeds
// usage, the pusher receives the meter totals, and the closer lists and
// finalizes draft invoices.
func WithDeployBilling(
	reader deploybilling.UsageReader,
	pusher billingmeter.Pusher,
	closer invoicecloser.Closer,
) Option {
	return func(o *harnessOpts) {
		o.billingUsageReader = reader
		o.billingPusher = pusher
		o.billingCloser = closer
	}
}

// New creates a new test harness with all services started and registered.
// All resources are automatically cleaned up when the test completes.
func New(t *testing.T, opts ...Option) *Harness {
	t.Helper()

	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	var o harnessOpts
	for _, opt := range opts {
		opt(&o)
	}
	if o.timeout == 0 {
		o.timeout = 120 * time.Second
	}
	if o.clock == nil {
		o.clock = clock.New()
	}

	start := time.Now()

	// Start the shared backing services in parallel. Restate starts after its
	// handlers are constructed because it is private to this test and is
	// registered as one complete worker deployment.
	var wg sync.WaitGroup
	var mysqlCfg containers.MySQLConfig
	var chCfg containers.ClickHouseConfig
	var testVault *vaulttestutil.TestVault

	wg.Add(3)

	go func() {
		defer wg.Done()
		s := time.Now()
		mysqlCfg = containers.MySQL(t)
		t.Logf("MySQL started in %s", time.Since(s))
	}()

	go func() {
		defer wg.Done()
		s := time.Now()
		chCfg = containers.ClickHouse(t)
		t.Logf("ClickHouse started in %s", time.Since(s))
	}()

	go func() {
		defer wg.Done()
		s := time.Now()
		testVault = vaulttestutil.StartTestVault(t)
		t.Logf("Vault started in %s", time.Since(s))
	}()

	wg.Wait()
	t.Logf("Backing containers started in %s", time.Since(start))

	// Connect to MySQL
	database, err := db.New(mysqlCfg.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	// Connect to ClickHouse
	chDSN := chCfg.DSN
	chClient, err := clickhouse.New(clickhouse.Config{
		URL: chDSN,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, chClient.Close()) })

	// Get direct connection for inserting test data
	chOpts, err := ch.ParseDSN(chDSN)
	require.NoError(t, err)
	conn, err := ch.Open(chOpts)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	// Create seeder for test data
	vaultClient := vault.NewConnectVaultServiceClient(testVault.Client)

	seeder := seed.New(t, database, vaultClient)

	// Unified cron service: every scheduled task runs as a handler on
	// hydra.v1.CronService. Heartbeats are noop in tests; the slack
	// webhook is empty so quota-check skips notification calls.
	cronSvc, err := cron.New(cron.Config{
		DB:                        database,
		Clickhouse:                chClient,
		Clock:                     o.clock,
		RatelimitDB:               ratelimitdb.New(database.RW(), database.RO()),
		SlackQuotaCheckWebhookURL: "",
		// Deploy billing is a no-op by default (nil reader + empty Stripe key);
		// WithDeployBilling injects fakes for tests that exercise the push/close.
		BillingUsageReader: o.billingUsageReader,
		BillingPusher:      o.billingPusher,
		BillingCloser:      o.billingCloser,
		StripeSecretKey:    "",
		// Deploy spend-check dependencies are empty in tests: no WorkOS/Resend
		// means the check resolves no recipients and logs instead of emailing.
		WorkOSAPIKey:   "",
		ResendAPIKey:   "",
		BillingBaseURL: "",
		Heartbeats: cron.Heartbeats{
			QuotaCheck:         healthcheck.NewNoop(),
			KeyRefill:          healthcheck.NewNoop(),
			KeyLastUsedSync:    healthcheck.NewNoop(),
			AuditLogExport:     healthcheck.NewNoop(),
			AuditLogCleanup:    healthcheck.NewNoop(),
			RatelimitCleanup:   healthcheck.NewNoop(),
			DeployBillingPush:  healthcheck.NewNoop(),
			DeployBillingClose: healthcheck.NewNoop(),
			DeploySpendCheck:   healthcheck.NewNoop(),
		},
	})
	require.NoError(t, err)

	clickhouseUserSvc := clickhouseuser.New(clickhouseuser.Config{
		DB:         database,
		Vault:      vaultClient,
		Clickhouse: chClient,
	})

	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)

	deploySvc, err := deploy.New(deploy.Config{
		DB:            database,
		Auditlogs:     auditlogSvc,
		Clickhouse:    chClient,
		DefaultDomain: "test.example.com",
		DashboardURL:  "https://app.unkey.com",
		Vault:         vaultClient,

		GitHub: nil,
		Build: deploy.BuildConfig{
			Backend:    deploy.BuildBackendDepot,
			Depot:      deploy.DepotConfig{APIUrl: "", ProjectRegion: "", ProjectPrefix: "builds-test"},
			Kubernetes: deploy.KubernetesBuildConfig{Namespace: "", Image: ""},
		},
		K8s:                             nil,
		BuildSteps:                      batch.NewNoop[schema.BuildStepV1](),
		BuildStepLogs:                   batch.NewNoop[schema.BuildStepLogV1](),
		RegistryConfig:                  deploy.RegistryConfig{Repository: "", Username: "", Password: "", Insecure: false},
		BuildPlatform:                   deploy.BuildPlatform{Platform: "", Architecture: ""},
		AllowUnauthenticatedDeployments: false,
		// Nil admin: a superseded sibling's row is still marked, its invocation
		// just keeps running. Enforcement off matches production's rollout state.
		RestateAdmin:      nil,
		EnforceDeployGate: false,
	})
	require.NoError(t, err)

	keyLastUsedPartitionSvc, err := keylastusedsync.NewPartitionService(keylastusedsync.PartitionConfig{
		DB:         database,
		Clickhouse: chClient,
	})
	require.NoError(t, err)

	// The build slot service audits slot occupancy against the Restate
	// admin API, and Teardown cancels in-flight Deploy invocations through
	// it, but the admin URL is only known after containers.Restate starts
	// below, and that start needs the constructed services. The lazy
	// adapter breaks the cycle: it is set directly after the container is
	// up, and no handler runs before that.
	lazyAdmin := &lazyInvocationLiveness{mu: sync.Mutex{}, client: nil}
	buildSlotSvc := buildslot.New(buildslot.Config{
		DB:           database,
		RestateAdmin: lazyAdmin,
	})

	// CheckWorkspaceSpend sends Teardown/Resume to this service. Restate
	// retries calls to unregistered services indefinitely, so it must be
	// registered here or a dispatched check never completes.
	teardownSvc, err := deployteardown.New(deployteardown.Config{
		DB:                database,
		Admin:             lazyAdmin,
		DrainPollInterval: 200 * time.Millisecond,
		DrainGraceTimeout: 2 * time.Second,
	})
	require.NoError(t, err)

	// Register every worker service as one deployment on this test's own
	// Restate. Use the proto-generated wrappers (same as run.go) to get
	// correct service names.
	restateCfg := containers.Restate(t,
		hydrav1.NewCronServiceServer(cronSvc),
		// The deploy billing orchestrator (push and close) fans out to this
		// per-workspace push service, so it must be bound for those handlers to
		// route end to end.
		hydrav1.NewDeployBillingPushServiceServer(cronSvc.DeployBillingPushServer()),
		hydrav1.NewDeploySpendCheckServiceServer(cronSvc.DeploySpendCheckServer()),
		hydrav1.NewClickhouseUserServiceServer(clickhouseUserSvc),
		hydrav1.NewKeyLastUsedPartitionServiceServer(keyLastUsedPartitionSvc),
		hydrav1.NewDeployServiceServer(deploySvc),
		hydrav1.NewDeployTeardownServiceServer(teardownSvc),
		hydrav1.NewBuildSlotServiceServer(buildSlotSvc),
	)
	lazyAdmin.set(restateadmin.New(restateadmin.Config{
		BaseURL: restateCfg.AdminURL,
		APIKey:  "",
	}))
	t.Logf("Total harness setup in %s", time.Since(start))

	// The timeout limits test operations, not container startup and service
	// readiness. Starting it before setup can return an already-expired context
	// to the first request when CI is under load.
	ctx, cancel := context.WithTimeout(context.Background(), o.timeout)
	t.Cleanup(cancel)

	return &Harness{
		Ctx:            ctx,
		DB:             database,
		Seed:           seeder,
		ClickHouseSeed: seed.NewClickHouseSeeder(t, conn),
		ClickHouse:     chClient,
		ClickHouseConn: conn,
		ClickHouseDSN:  chDSN,
		VaultClient:    vaultClient,
		VaultToken:     testVault.Token,
		Restate:        restateCfg.IngressClient,
		RestateAdmin:   restateCfg.AdminURL,
		Clock:          o.clock,
	}
}

// lazyInvocationLiveness defers the Restate admin client until the test
// container is running. See the comment at the buildslot.New call site.
type lazyInvocationLiveness struct {
	mu     sync.Mutex
	client *restateadmin.Client
}

var _ buildslot.InvocationLiveness = (*lazyInvocationLiveness)(nil)
var _ deployteardown.InvocationCanceler = (*lazyInvocationLiveness)(nil)

func (l *lazyInvocationLiveness) set(client *restateadmin.Client) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.client = client
}

func (l *lazyInvocationLiveness) FindLiveInvocations(ctx context.Context, invocationIDs []string) (map[string]bool, error) {
	l.mu.Lock()
	client := l.client
	l.mu.Unlock()
	if client == nil {
		return nil, errors.New("restate admin client not initialized yet")
	}
	return client.FindLiveInvocations(ctx, invocationIDs)
}

func (l *lazyInvocationLiveness) CancelInvocation(ctx context.Context, invocationID string) error {
	l.mu.Lock()
	client := l.client
	l.mu.Unlock()
	if client == nil {
		return errors.New("restate admin client not initialized yet")
	}
	return client.CancelInvocation(ctx, invocationID)
}
