package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	restate "github.com/restatedev/sdk-go"
	restateServer "github.com/restatedev/sdk-go/server"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/dockertest"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type webhookHarnessConfig struct {
	Services      []restate.ServiceDefinition
	WebhookSecret string
}

type webhookHarness struct {
	ctx     context.Context
	CtrlURL string
	DB      db.Database
	Seed    *seed.Seeder
	Secret  string
}

func newWebhookHarness(t *testing.T, cfg webhookHarnessConfig) *webhookHarness {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)

	restateCfg := dockertest.Restate(t)

	restateSrv := restateServer.NewRestate().WithLogger(logging.Handler(), false)
	for _, service := range cfg.Services {
		restateSrv.Bind(service)
	}

	restateHandler, err := restateSrv.Handler()
	require.NoError(t, err)

	workerMux := http.NewServeMux()
	workerMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	workerMux.Handle("/", restateHandler)

	workerListener, err := net.Listen("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	workerServer := httptest.NewUnstartedServer(h2c.NewHandler(workerMux, &http2.Server{}))
	workerServer.Listener = workerListener
	workerServer.Start()
	t.Cleanup(workerServer.Close)

	workerPort := workerListener.Addr().(*net.TCPAddr).Port
	registration := &restateRegistration{adminURL: restateCfg.AdminURL, registerAs: fmt.Sprintf("http://%s:%d", dockerHost(), workerPort)}
	require.NoError(t, registration.register(ctx))

	mysqlCfg := dockertest.MySQL(t)
	database, err := db.New(db.Config{
		Logger:      logging.NewNoop(),
		PrimaryDSN:  mysqlCfg.DSN,
		ReadOnlyDSN: "",
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	seeder := seed.New(t, database, nil)
	seeder.Seed(ctx)

	ctrlAddr := pickAddr(t)
	ctrlPort := ctrlAddr.Port

	secret := cfg.WebhookSecret
	if secret == "" {
		secret = uid.New("whsec")
	}

	apiConfig := Config{
		InstanceID:            "test",
		Region:                "local",
		HttpPort:              ctrlPort,
		PrometheusPort:        0,
		DatabasePrimary:       mysqlCfg.DSN,
		OtelEnabled:           false,
		OtelTraceSamplingRate: 0,
		TLSConfig:             nil,
		AuthToken:             "",
		Restate: RestateConfig{
			URL:    restateCfg.IngressURL,
			APIKey: "",
		},
		AvailableRegions:    []string{"local.dev"},
		GitHubWebhookSecret: secret,
		DefaultDomain:       "",
		RegionalDomain:      "",
	}

	ctrlCtx, ctrlCancel := context.WithCancel(ctx)
	t.Cleanup(ctrlCancel)

	go func() {
		require.NoError(t, Run(ctrlCtx, apiConfig))
	}()

	ctrlURL := fmt.Sprintf("http://127.0.0.1:%d", ctrlPort)
	require.Eventually(t, func() bool {
		resp, err := http.Get(ctrlURL + "/health")
		if err != nil {
			return false
		}
		defer func() { _ = resp.Body.Close() }()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 200*time.Millisecond)

	return &webhookHarness{
		ctx:     ctx,
		CtrlURL: ctrlURL,
		DB:      database,
		Seed:    seeder,
		Secret:  secret,
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
	return []connect.ClientOption{}
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

type restateRegistration struct {
	adminURL   string
	registerAs string
}

func (r *restateRegistration) register(ctx context.Context) error {
	registerURL := r.adminURL + "/deployments"
	payload := []byte("{\"uri\": \"" + r.registerAs + "\"}")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, registerURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	requireJSON(req)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmtStatus(resp.StatusCode)
	}
	return nil
}

func requireJSON(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
}

type statusErr struct {
	code int
}

func (e statusErr) Error() string {
	return fmt.Sprintf("unexpected status code: %d", e.code)
}

func fmtStatus(code int) error {
	return statusErr{code: code}
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

func dockerHost() string {
	if runtime.GOOS == "darwin" {
		return "host.docker.internal"
	}
	return "172.17.0.1"
}
