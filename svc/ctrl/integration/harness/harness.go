// Package harness provides a unified test harness for ctrl worker integration tests.
// It starts all required services (MySQL, ClickHouse, Restate, Vault) and registers
// all worker handlers, enabling end-to-end testing of any handler without per-test setup.
package harness

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/restatedev/sdk-go/ingress"
	restateServer "github.com/restatedev/sdk-go/server"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/dockertest"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/worker/clickhouseuser"
	"github.com/unkeyed/unkey/svc/ctrl/worker/quotacheck"
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

	// Seed provides methods to create test entities in the database.
	Seed *seed.Seeder

	// ClickHouse is the ClickHouse client for analytics queries.
	ClickHouse clickhouse.ClickHouse

	// ClickHouseConn is a direct ClickHouse connection for inserting test data.
	ClickHouseConn ch.Conn

	// ClickHouseDSN is the ClickHouse connection string.
	ClickHouseDSN string

	// VaultClient is a real vault client for encryption/decryption.
	VaultClient vaultv1connect.VaultServiceClient

	// VaultToken is the bearer token for the vault service.
	VaultToken string

	// Restate is the ingress client for calling Restate services.
	Restate *ingress.Client

	// RestateIngress is the URL for calling Restate handlers.
	RestateIngress string

	// RestateAdmin is the URL for Restate admin operations.
	RestateAdmin string
}

// New creates a new test harness with all services started and registered.
// All resources are automatically cleaned up when the test completes.
func New(t *testing.T) *Harness {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	t.Cleanup(cancel)

	// Start Restate container
	restateCfg := dockertest.Restate(t)

	// Start MySQL container
	mysqlCfg := dockertest.MySQL(t)
	database, err := db.New(db.Config{
		Logger:      logging.NewNoop(),
		PrimaryDSN:  mysqlCfg.DSN,
		ReadOnlyDSN: "",
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	// Start ClickHouse container
	chCfg := dockertest.ClickHouse(t)
	chDSN := chCfg.DSN
	chClient, err := clickhouse.New(clickhouse.Config{
		URL:    chDSN,
		Logger: logging.NewNoop(),
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, chClient.Close()) })

	// Get direct connection for inserting test data
	opts, err := ch.ParseDSN(chDSN)
	require.NoError(t, err)
	conn, err := ch.Open(opts)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	// Start real vault
	testVault := vaulttestutil.StartTestVault(t)

	// Create seeder for test data
	seeder := seed.New(t, database, testVault.Client)

	// Create all services
	quotaCheckSvc, err := quotacheck.New(quotacheck.Config{
		DB:              database,
		Clickhouse:      chClient,
		Logger:          logging.NewNoop(),
		Heartbeat:       healthcheck.NewNoop(),
		SlackWebhookURL: "",
	})
	require.NoError(t, err)

	clickhouseUserSvc := clickhouseuser.New(clickhouseuser.Config{
		DB:         database,
		Vault:      testVault.Client,
		Clickhouse: chClient,
		Logger:     logging.NewNoop(),
	})

	// Set up Restate server with all services
	// Use the proto-generated wrappers (same as run.go) to get correct service names
	restateSrv := restateServer.NewRestate().WithLogger(logging.Handler(), false)
	restateSrv.Bind(hydrav1.NewQuotaCheckServiceServer(quotaCheckSvc))
	restateSrv.Bind(hydrav1.NewClickhouseUserServiceServer(clickhouseUserSvc))

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

	adminClient := restateadmin.New(restateadmin.Config{BaseURL: restateCfg.AdminURL})
	require.NoError(t, adminClient.RegisterDeployment(ctx, registerAs))

	return &Harness{
		Ctx:            ctx,
		DB:             database,
		Seed:           seeder,
		ClickHouse:     chClient,
		ClickHouseConn: conn,
		ClickHouseDSN:  chDSN,
		VaultClient:    testVault.Client,
		VaultToken:     testVault.Token,
		Restate:        ingress.NewClient(restateCfg.IngressURL),
		RestateIngress: restateCfg.IngressURL,
		RestateAdmin:   restateCfg.AdminURL,
	}
}

// dockerHost returns the hostname to use for connecting from Docker containers.
func dockerHost() string {
	if runtime.GOOS == "darwin" {
		return "host.docker.internal"
	}
	return "172.17.0.1"
}
