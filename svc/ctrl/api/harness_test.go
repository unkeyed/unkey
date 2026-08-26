package api

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/rpc/interceptor"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deploymentcreate"
	"golang.org/x/net/http2"
)

type webhookHarnessConfig struct {
	Services []restate.ServiceDefinition
	// DBServices builds Restate services that need the harness database.
	// Nil binds the real DeploymentCreateService; a test needing a variant
	// (e.g. a failing audit sink) supplies its own and replaces the default.
	DBServices        func(db.Database) []restate.ServiceDefinition
	WebhookSecret     string
	EnforceDeployGate bool
}

type webhookHarness struct {
	ctx        context.Context
	CtrlURL    string
	DB         db.Database
	Seed       *seed.Seeder
	Secret     string
	AuthToken  string
	IngressURL string
}

func newWebhookHarness(t *testing.T, cfg webhookHarnessConfig) *webhookHarness {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)

	mysqlCfg := containers.MySQL(t)
	database, err := db.New(mysqlCfg.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	services := cfg.Services
	if cfg.DBServices != nil {
		services = append(services, cfg.DBServices(database)...)
	} else {
		auditlogSvc, auditErr := auditlogs.New(auditlogs.Config{DB: database})
		require.NoError(t, auditErr)
		services = append(services, hydrav1.NewDeploymentCreateServiceServer(
			deploymentcreate.New(deploymentcreate.Config{
				DB:           database,
				Auditlogs:    auditlogSvc,
				RestateAdmin: nil,
			}),
			restate.WithIdempotencyRetention(12*time.Hour),
		))
	}

	restateCfg := containers.Restate(t, services...)

	seeder := seed.New(t, database, nil)
	seeder.Seed(ctx)

	ctrlAddr := pickAddr(t)
	ctrlPort := ctrlAddr.Port

	secret := cfg.WebhookSecret
	if secret == "" {
		secret = uid.New("whsec")
	}

	authToken := uid.New("ctrl_test")
	apiConfig := Config{
		InstanceID:     "test",
		Region:         "local",
		HttpPort:       ctrlPort,
		PrometheusPort: 0,
		AuthToken:      authToken,

		DefaultDomain:  "",
		RegionalDomain: "",
		Database:       mysqlCfg.DSN,
		Observability:  config.Observability{},
		Restate: RestateConfig{
			URL:    restateCfg.IngressURL,
			APIKey: "",
		},
		GitHub: GitHubConfig{
			WebhookSecret: secret,
		},
		DeployGate: DeployGateConfig{
			Enforce: cfg.EnforceDeployGate,
		},
	}

	ctrlCtx, ctrlCancel := context.WithCancel(ctx)
	runErr := make(chan error, 1)
	go func() {
		runErr <- Run(ctrlCtx, apiConfig)
	}()
	t.Cleanup(func() {
		ctrlCancel()
		require.NoError(t, <-runErr)
	})

	ctrlURL := fmt.Sprintf("http://127.0.0.1:%d", ctrlPort)
	require.Eventually(t, func() bool {
		resp, err := http.Get(ctrlURL + "/health/live")
		if err != nil {
			return false
		}
		defer func() { _ = resp.Body.Close() }()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 200*time.Millisecond)

	return &webhookHarness{
		ctx:        ctx,
		CtrlURL:    ctrlURL,
		DB:         database,
		Seed:       seeder,
		Secret:     secret,
		AuthToken:  authToken,
		IngressURL: restateCfg.IngressURL,
	}
}

func (h *webhookHarness) ConnectClient() *http.Client {
	if !strings.HasPrefix(h.CtrlURL, "http://") {
		return &http.Client{Timeout: 10 * time.Second}
	}

	return &http.Client{
		Timeout: 10 * time.Second,
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
}

func (h *webhookHarness) ConnectOptions() []connect.ClientOption {
	return []connect.ClientOption{
		connect.WithInterceptors(interceptor.NewHeaderInjector(map[string]string{
			"Authorization": "Bearer " + h.AuthToken,
		})),
	}
}

func (h *webhookHarness) RequestContext() context.Context {
	return context.Background()
}

func (h *webhookHarness) CreateProject(ctx context.Context, req seed.CreateProjectRequest) db.Project {
	return h.Seed.CreateProject(ctx, req)
}

func (h *webhookHarness) CreateEnvironment(ctx context.Context, req seed.CreateEnvironmentRequest) db.Environment {
	return h.Seed.CreateEnvironment(ctx, req)
}

func (h *webhookHarness) CreateApp(ctx context.Context, req seed.CreateAppRequest) db.App {
	return h.Seed.CreateApp(ctx, req)
}

func (h *webhookHarness) CreateAppWithSettings(ctx context.Context, req seed.CreateAppRequest, environmentID string) db.App {
	return h.Seed.CreateAppWithSettings(ctx, req, environmentID)
}

type addrInfo struct {
	Host string
	Port int
}

func pickAddr(t *testing.T) addrInfo {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { require.NoError(t, listener.Close()) }()

	addr, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok)

	return addrInfo{Host: addr.IP.String(), Port: addr.Port}
}
