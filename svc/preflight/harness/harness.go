// Package harness stands up a complete preflight environment against
// which both the dev-target binary and every probe unit test can run.
//
// It wraps pkg/dockertest (MySQL, ClickHouse, Restate), svc/vault
// testutil (Vault), svc/ctrl/api.Run (the ctrl HTTP API in-process),
// and svc/ctrl/integration/seed (preflight workspace/project/app/env).
// Caller passes a *testing.T; every container and goroutine is cleaned
// up automatically.
//
// The harness is deliberately standalone from svc/ctrl/integration/harness.
// That harness exists for Restate-worker integration tests and mocks the
// ctrl HTTP boundary; preflight needs the real ctrl API so webhook-to-
// deployment flows can be exercised end to end.
//
// Usage from a probe unit test:
//
//	func TestMyProbe(t *testing.T) {
//	    h := harness.Start(t, harness.Config{SeedPreflightProject: true})
//	    env := h.Env()
//	    res := (&probes.MyProbe{}).Run(context.Background(), env)
//	    require.True(t, res.OK)
//	}
//
// Usage from the dev-target test:
//
//	go test -run TestDev ./cmd/preflight/harness/...
//
// See the README in cmd/preflight for how to wire docker-compose.test
// before running.
package harness

import (
	"bytes"
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	restateingress "github.com/restatedev/sdk-go/ingress"
	restateServer "github.com/restatedev/sdk-go/server"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/dockertest"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/api"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/preflight/core"
	vaulttestutil "github.com/unkeyed/unkey/svc/vault/testutil"
)

// Config controls which pieces of the harness come up. Zero value is a
// bare environment (databases + vault only); opt in to the expensive
// bits on a per-test basis to keep fast paths fast.
type Config struct {
	// SeedPreflightProject pre-creates a workspace/project/app/environment
	// inside the seeded database, mimicking the dedicated "Unkey Preflight"
	// tenant that real runs live in. Probes that exercise CreateDeployment
	// against a known project set this to true.
	SeedPreflightProject bool

	// MockRestateServices are Restate service implementations the harness
	// registers on its in-process worker. Tests that want to assert on
	// deploy-workflow invocations provide a mock DeployService here and
	// read its recorded requests.
	//
	// Leave nil to run without any restate worker; calls that hit
	// Restate will queue but not execute.
	MockRestateServices []restate.ServiceDefinition

	// WebhookSecret overrides the randomly-generated GitHub webhook
	// secret. Probes that sign their own payloads typically leave this
	// empty and read h.WebhookSecret afterwards.
	WebhookSecret string
}

// Harness is the running environment. Fields are populated based on the
// Config supplied to Start; unused fields are zero-valued.
//
// Every field on Harness is safe to use in parallel across probes, with
// the exception of the Seeder helper methods which serialise at the DB
// level but do not coordinate.
type Harness struct {
	// CtrlURL is the HTTP base URL of the in-process ctrl API.
	CtrlURL string

	// AuthToken is the bearer token the ctrl API accepts.
	AuthToken string

	// WebhookSecret is the GitHub webhook HMAC secret the ctrl API is
	// configured with. Copy it into synthetic webhook payloads so the
	// verifier accepts them.
	WebhookSecret string

	// Region is the region label the ctrl API logs with. Always "local"
	// for harness runs.
	Region string

	// DB is the MySQL connection shared with the in-process ctrl API.
	DB db.Database

	// ClickHouse is the analytics client pointed at the shared CH test
	// container's `default` database.
	ClickHouse clickhouse.ClickHouse

	// ClickHouseConn is a direct ClickHouse driver connection for tests
	// that need to insert raw rows.
	ClickHouseConn ch.Conn

	// Vault is a real Vault RPC client backed by an in-process test Vault.
	Vault vault.VaultServiceClient

	// VaultToken is the bearer token for Vault RPC.
	VaultToken string

	// Restate is the ingress client for invoking services registered on
	// the shared test Restate container.
	Restate *restateingress.Client

	// RestateIngress is the URL callers pass when constructing their own
	// clients. Same URL the ctrl API is configured to speak to.
	RestateIngress string

	// RestateAdmin is the admin URL for registering additional Restate
	// deployments.
	RestateAdmin string

	// Seed is the integration seeder. Call h.Seed.CreateProject / CreateApp
	// / CreateEnvironment when a test needs extra entities beyond the
	// preflight project.
	Seed *seed.Seeder

	// Project, App, Environment are populated when Config.SeedPreflightProject
	// is true. They represent the "Unkey Preflight" tenant.
	Project     *db.Project
	App         *db.App
	Environment *db.Environment
}

// Env builds a *core.Env ready to hand to a probe. Populates only the
// fields the harness can back. Probes that need clients the harness
// does not provide assert for themselves and short-circuit.
func (h *Harness) Env() *core.Env {
	env := &core.Env{
		Target:                   core.TargetDev,
		Region:                   h.Region,
		RunID:                    uid.New("pflt"),
		CtrlBaseURL:              h.CtrlURL,
		CtrlAuthToken:            h.AuthToken,
		GitHubWebhookSecret:      h.WebhookSecret,
		PreflightProjectID:       "",
		PreflightAppID:           "",
		PreflightEnvironmentSlug: "",
		PreflightProjectSlug:     "",
		PreflightAppSlug:         "",
		PreflightWorkspaceSlug:   "",
		PreflightApex:            "",
		GitHubAppID:              0,
		GitHubInstallationID:     0,
		GitHubPrivateKeyPEM:      "",
		PreflightTestRepo:        "",
		DB:                       h.DB,
		ClickHouse:               h.ClickHouse,
	}

	if h.Project != nil {
		env.PreflightProjectID = h.Project.ID
		env.PreflightProjectSlug = h.Project.Slug
	}

	if h.App != nil {
		env.PreflightAppID = h.App.ID
		env.PreflightAppSlug = h.App.Slug
	}

	if h.Environment != nil {
		env.PreflightEnvironmentSlug = h.Environment.Slug
	}

	if ws := h.Seed.Resources.UserWorkspace; ws.ID != "" {
		env.PreflightWorkspaceSlug = ws.Slug
	}

	return env
}

// Start brings up the harness. Every container and goroutine is
// registered with t.Cleanup; callers do not need to Stop() anything.
//
// Steps:
//  1. Start MySQL / ClickHouse / Restate / Vault containers in parallel.
//  2. If the caller provided MockRestateServices, stand up a Restate
//     worker on a random port and register it with Restate admin.
//  3. Start the ctrl API via svc/ctrl/api.Run pointed at the databases
//     and Restate. Wait for /health/live to go green.
//  4. Seed a root workspace (so the seeder is initialised); if
//     Config.SeedPreflightProject is set, additionally create the
//     preflight project/app/environment.
func Start(t *testing.T, cfg Config) *Harness {
	t.Helper()

	// 300s handles cold-start variance: MySQL 15s + ClickHouse 10s +
	// Restate 20s + admin-registration retry loop up to 60s + seed +
	// ctrl API startup. Steady-state harness tests finish in ~20s; the
	// large budget only matters when containers are truly cold.
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	t.Cleanup(cancel)

	started := time.Now()

	// Containers in parallel; the ctrl API cannot start until they're up.
	mysqlCfg := dockertest.MySQL(t)
	chCfg := dockertest.ClickHouse(t)
	restateCfg := dockertest.Restate(t)
	testVault := vaulttestutil.StartTestVault(t)

	// Connect MySQL.
	database, err := db.New(db.Config{
		PrimaryDSN:  mysqlCfg.DSN,
		ReadOnlyDSN: "",
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	// Connect ClickHouse.
	chClient, err := clickhouse.New(clickhouse.Config{URL: chCfg.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, chClient.Close()) })

	chOpts, err := ch.ParseDSN(chCfg.DSN)
	require.NoError(t, err)
	chConn, err := ch.Open(chOpts)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, chConn.Close()) })

	// Vault client.
	vaultClient := vault.NewConnectVaultServiceClient(testVault.Client)

	// Seeder. Always Seed() the root workspace so the DB has the
	// fixtures every ctrl-plane endpoint expects.
	seeder := seed.New(t, database, vaultClient)
	seeder.Seed(ctx)

	// Optional: mock Restate worker. Caller supplies services they care
	// about (typically a DeployService stub that captures requests).
	if len(cfg.MockRestateServices) > 0 {
		registerMockWorker(t, ctx, restateCfg.AdminURL, cfg.MockRestateServices)
	}

	// ctrl API in-process.
	webhookSecret := cfg.WebhookSecret
	if webhookSecret == "" {
		webhookSecret = uid.New("whsec")
	}

	authToken := uid.New("ctrl_test")

	ctrlAddr := pickAddr(t)
	apiCfg := api.Config{
		InstanceID:     "preflight-harness",
		Region:         "local",
		HttpPort:       ctrlAddr.Port,
		PrometheusPort: 0,
		AuthToken:      authToken,
		DefaultDomain:  "",
		RegionalDomain: "",
		CnameDomain:    "",
		Database: config.DatabaseConfig{
			Primary:         mysqlCfg.DSN,
			ReadonlyReplica: "",
		},
		Observability: config.Observability{}, //nolint:exhaustruct // defaults are fine for the harness
		Restate: api.RestateConfig{
			URL:      restateCfg.IngressURL,
			AdminURL: restateCfg.AdminURL,
			APIKey:   "",
		},
		GitHub: api.GitHubConfig{
			WebhookSecret: webhookSecret,
			AppID:         0,
			PrivateKeyPEM: "",
		},
		DomainConnect: api.DomainConnectConfig{PrivateKeyPEM: ""},
	}

	apiCtx, apiCancel := context.WithCancel(ctx)
	t.Cleanup(apiCancel)

	go func() {
		if err := api.Run(apiCtx, apiCfg); err != nil && !isExpectedShutdown(err) {
			logger.Error("preflight harness: ctrl API exited", "error", err.Error())
		}
	}()

	ctrlURL := fmt.Sprintf("http://127.0.0.1:%d", ctrlAddr.Port)
	waitForCtrlHealthy(t, ctrlURL)

	h := &Harness{
		CtrlURL:        ctrlURL,
		AuthToken:      authToken,
		WebhookSecret:  webhookSecret,
		Region:         apiCfg.Region,
		DB:             database,
		ClickHouse:     chClient,
		ClickHouseConn: chConn,
		Vault:          vaultClient,
		VaultToken:     testVault.Token,
		Restate:        restateingress.NewClient(restateCfg.IngressURL),
		RestateIngress: restateCfg.IngressURL,
		RestateAdmin:   restateCfg.AdminURL,
		Seed:           seeder,
		Project:        nil,
		App:            nil,
		Environment:    nil,
	}

	if cfg.SeedPreflightProject {
		h.seedPreflightProject(ctx)
	}

	logger.Info("preflight harness ready",
		"duration_ms", time.Since(started).Milliseconds(),
		"ctrl_url", ctrlURL,
		"seeded_project", cfg.SeedPreflightProject,
	)
	return h
}

func (h *Harness) seedPreflightProject(ctx context.Context) {
	ws := h.Seed.Resources.UserWorkspace
	envID := uid.New("env")

	project := h.Seed.CreateProject(ctx, seed.CreateProjectRequest{
		ID:               uid.New("prj"),
		WorkspaceID:      ws.ID,
		Name:             "preflight",
		Slug:             "preflight",
		DeleteProtection: false,
	})
	app := h.Seed.CreateAppWithSettings(ctx, seed.CreateAppRequest{
		ID:            uid.New("app"),
		WorkspaceID:   ws.ID,
		ProjectID:     project.ID,
		Name:          "testapp",
		Slug:          "testapp",
		DefaultBranch: "main",
	}, envID)
	environment := h.Seed.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:               envID,
		WorkspaceID:      ws.ID,
		ProjectID:        project.ID,
		AppID:            app.ID,
		Slug:             "production",
		Description:      "",
		SentinelConfig:   []byte("{}"),
		DeleteProtection: false,
	})

	h.Project = &project
	h.App = &app
	h.Environment = &environment
}

// registerMockWorker stands up an in-process Restate SDK worker serving
// the provided mock services and registers it with Restate admin so the
// ctrl API's invocations route to it.
func registerMockWorker(t *testing.T, ctx context.Context, adminURL string, services []restate.ServiceDefinition) {
	t.Helper()

	srv := restateServer.NewRestate().WithLogger(logger.GetHandler(), false)
	for _, s := range services {
		srv.Bind(s)
	}
	handler, err := srv.Handler()
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("/", handler)

	//nolint:gosec // test-only listener bound to all interfaces because Docker reaches it over host networking
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	require.NoError(t, err)

	//nolint:exhaustruct // default http2.Server settings are fine for the test worker
	workerSrv := httptest.NewUnstartedServer(h2c.NewHandler(mux, &http2.Server{}))
	workerSrv.Listener = listener
	workerSrv.Start()
	t.Cleanup(workerSrv.Close)

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok)
	registerAs := fmt.Sprintf("http://%s:%d", dockerHost(), tcpAddr.Port)

	payload := []byte(fmt.Sprintf(`{"uri": %q}`, registerAs))
	// Restate admin is usually ready when the dockertest TCP probe
	// succeeds, but under concurrent test load it sometimes drags.
	// Retry on network-level failures and 5xx responses; abort only
	// when ctx is cancelled.
	//nolint:exhaustruct // the default transport is correct for a loopback admin call
	client := &http.Client{Timeout: 30 * time.Second}
	deadline := time.Now().Add(60 * time.Second)

	var lastErr error
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, adminURL+"/deployments", bytes.NewReader(payload))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(500 * time.Millisecond)
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode/100 == 2 {
			return
		}
		lastErr = fmt.Errorf("restate admin returned %d", resp.StatusCode)
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("preflight harness: restate register failed within 60s: %v", lastErr)
}

func waitForCtrlHealthy(t *testing.T, ctrlURL string) {
	t.Helper()
	//nolint:exhaustruct // default HTTP client is appropriate for a liveness poll
	client := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, network, addr)
			},
			ReadIdleTimeout: 10 * time.Second,
			PingTimeout:     5 * time.Second,
		},
	}
	require.Eventually(t, func() bool {
		resp, err := client.Get(ctrlURL + "/health/live")
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 200*time.Millisecond, "ctrl API never became healthy at %s", ctrlURL)
}

// ---------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------

type addrInfo struct {
	Host string
	Port int
}

func pickAddr(t *testing.T) addrInfo {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { require.NoError(t, listener.Close()) }()

	addr, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok)
	return addrInfo{Host: addr.IP.String(), Port: addr.Port}
}

func dockerHost() string {
	if runtime.GOOS == "darwin" {
		return "host.docker.internal"
	}
	return "172.17.0.1"
}

func isExpectedShutdown(err error) bool {
	// api.Run returns when its context is cancelled; treat the
	// well-known suspects as "harness shutting down normally".
	if err == nil {
		return true
	}
	return err == context.Canceled || err == sql.ErrConnDone || err == http.ErrServerClosed
}
