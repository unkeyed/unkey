// Package harness provides a unified test harness for ctrl worker integration tests.
// It starts all required services (MySQL, ClickHouse, Restate, Vault) and registers
// all worker handlers, enabling end-to-end testing of any handler without per-test setup.
package harness

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/restatedev/sdk-go/ingress"
	restateServer "github.com/restatedev/sdk-go/server"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	ratelimitdb "github.com/unkeyed/unkey/internal/services/ratelimit/db"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/worker/buildslot"
	"github.com/unkeyed/unkey/svc/ctrl/worker/clickhouseuser"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deploy"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deployment"
	"github.com/unkeyed/unkey/svc/ctrl/worker/keylastusedsync"
	vaulttestutil "github.com/unkeyed/unkey/svc/vault/testutil"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
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

	// RestateIngress is the URL for calling Restate handlers.
	RestateIngress string

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
	diskMySQL bool
	timeout   time.Duration
	clock     clock.Clock
}

// WithDiskMySQL starts MySQL with disk-backed storage instead of the default
// 256MB tmpfs. Use this for performance tests with large datasets.
func WithDiskMySQL() Option {
	return func(o *harnessOpts) {
		o.diskMySQL = true
	}
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

	ctx, cancel := context.WithTimeout(context.Background(), o.timeout)
	t.Cleanup(cancel)

	start := time.Now()

	// Start all containers in parallel
	var wg sync.WaitGroup
	var restateCfg containers.RestateConfig
	var mysqlCfg containers.MySQLConfig
	var chCfg containers.ClickHouseConfig
	var testVault *vaulttestutil.TestVault

	wg.Add(4)

	go func() {
		defer wg.Done()
		s := time.Now()
		restateCfg = containers.Restate(t)
		t.Logf("Restate started in %s", time.Since(s))
	}()

	go func() {
		defer wg.Done()
		s := time.Now()
		var mysqlOpts []containers.MySQLOpt
		if o.diskMySQL {
			mysqlOpts = append(mysqlOpts, containers.WithDiskStorage())
		}
		mysqlCfg = containers.MySQL(t, mysqlOpts...)
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
	t.Logf("All containers started in %s", time.Since(start))

	// Connect to MySQL
	database, err := db.New(db.Config{
		PrimaryDSN:  mysqlCfg.DSN,
		ReadOnlyDSN: "",
	})
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
		// Deploy billing push disabled in tests: nil reader + empty Stripe key
		// make the handler a no-op.
		BillingUsageReader: nil,
		StripeSecretKey:    "",
		Heartbeats: cron.Heartbeats{
			QuotaCheck:        healthcheck.NewNoop(),
			KeyRefill:         healthcheck.NewNoop(),
			KeyLastUsedSync:   healthcheck.NewNoop(),
			AuditLogExport:    healthcheck.NewNoop(),
			DeployBillingPush: healthcheck.NewNoop(),
		},
	})
	require.NoError(t, err)

	clickhouseUserSvc := clickhouseuser.New(clickhouseuser.Config{
		DB:         database,
		Vault:      vaultClient,
		Clickhouse: chClient,
	})

	deploySvc := deploy.New(deploy.Config{
		DB:            database,
		Clickhouse:    chClient,
		DefaultDomain: "test.example.com",
		DashboardURL:  "https://app.unkey.com",
		Vault:         vaultClient,

		GitHub:                          nil,
		DepotConfig:                     deploy.DepotConfig{APIUrl: "", ProjectRegion: "", ProjectPrefix: "builds-test"},
		BuildSteps:                      batch.NewNoop[schema.BuildStepV1](),
		BuildStepLogs:                   batch.NewNoop[schema.BuildStepLogV1](),
		RegistryConfig:                  deploy.RegistryConfig{Repository: "", Username: "", Password: ""},
		BuildPlatform:                   deploy.BuildPlatform{Platform: "", Architecture: ""},
		AllowUnauthenticatedDeployments: false,
	})

	keyLastUsedPartitionSvc, err := keylastusedsync.NewPartitionService(keylastusedsync.PartitionConfig{
		DB:         database,
		Clickhouse: chClient,
	})
	require.NoError(t, err)

	deploymentSvc := deployment.New(deployment.Config{
		DB: database,
	})

	buildSlotSvc := buildslot.New(buildslot.Config{
		DB: database,
	})

	// Set up Restate server with all services
	// Use the proto-generated wrappers (same as run.go) to get correct service names
	restateSrv := restateServer.NewRestate()
	restateSrv.Bind(hydrav1.NewCronServiceServer(cronSvc))
	restateSrv.Bind(hydrav1.NewClickhouseUserServiceServer(clickhouseUserSvc))
	restateSrv.Bind(hydrav1.NewKeyLastUsedPartitionServiceServer(keyLastUsedPartitionSvc))
	restateSrv.Bind(hydrav1.NewDeployServiceServer(deploySvc))
	restateSrv.Bind(hydrav1.NewDeploymentServiceServer(deploymentSvc))
	restateSrv.Bind(hydrav1.NewBuildSlotServiceServer(buildSlotSvc))

	restateHandler, err := restateSrv.Handler()
	require.NoError(t, err)

	workerMux := http.NewServeMux()
	workerMux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	workerMux.Handle("/", restateHandler)

	workerListener, err := net.Listen("tcp", "0.0.0.0:0") //nolint:gosec // Test server needs to bind all interfaces for Docker access
	require.NoError(t, err)
	workerServer := httptest.NewUnstartedServer(h2c.NewHandler(workerMux, &http2.Server{})) //nolint:exhaustruct // Only need default settings for test
	workerServer.Listener = workerListener
	workerServer.Start()
	t.Cleanup(workerServer.Close)

	tcpAddr, ok := workerListener.Addr().(*net.TCPAddr)
	require.True(t, ok, "listener address must be TCP")
	workerPort := tcpAddr.Port
	registerAs := fmt.Sprintf("http://%s:%d", dockerHost(), workerPort)

	adminClient := restateadmin.New(restateadmin.Config{BaseURL: restateCfg.AdminURL, APIKey: ""})
	require.NoError(t, adminClient.RegisterDeployment(ctx, registerAs))
	t.Logf("Total harness setup in %s", time.Since(start))

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
		Restate:        ingress.NewClient(restateCfg.IngressURL),
		RestateIngress: restateCfg.IngressURL,
		RestateAdmin:   restateCfg.AdminURL,
		Clock:          o.clock,
	}
}

// dockerHost returns the hostname to use for connecting from Docker containers.
func dockerHost() string {
	if runtime.GOOS == "darwin" {
		return "host.docker.internal"
	}
	return "172.17.0.1"
}
