package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
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
	"github.com/restatedev/sdk-go/ingress"
	restateServer "github.com/restatedev/sdk-go/server"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/rpc/interceptor"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

const (
	readinessServiceName = "harnessReadiness"
	readinessHandlerName = "ping"
)

type webhookHarnessConfig struct {
	Services      []restate.ServiceDefinition
	WebhookSecret string
}

type webhookHarness struct {
	ctx       context.Context
	CtrlURL   string
	DB        db.Database
	Seed      *seed.Seeder
	Secret    string
	AuthToken string
}

func newWebhookHarness(t *testing.T, cfg webhookHarnessConfig) *webhookHarness {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)

	restateCfg := containers.Restate(t)

	restateSrv := restateServer.NewRestate().WithLogger(logger.GetHandler(), false)
	for _, service := range cfg.Services {
		restateSrv.Bind(service)
	}

	// Readiness probe object. Registering a deployment (register below) only
	// confirms Restate discovered the worker, not that its partition processor
	// can route invocations to it yet. Until it can, a Send is accepted but
	// never dispatched, so the first test would time out waiting for its
	// invocation. This is a virtual object, matching the keyed services under
	// test: on cold start keyed routing becomes ready later than unkeyed
	// service routing, so probing a plain service would pass too early. We
	// invoke it synchronously after registration and only return once it
	// responds, proving keyed invocations actually reach this worker.
	restateSrv.Bind(restate.NewObject(readinessServiceName).
		Handler(readinessHandlerName, restate.NewObjectHandler(
			func(_ restate.ObjectContext, in string) (string, error) { return in, nil })))

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
	deploymentID, err := registration.register(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { registration.deregister(context.Background(), deploymentID) })

	ingressClient := ingress.NewClient(restateCfg.IngressURL)
	require.Eventually(t, func() bool {
		reqCtx, reqCancel := context.WithTimeout(ctx, 2*time.Second)
		defer reqCancel()
		_, err := ingress.Object[string, string](ingressClient, readinessServiceName, "probe", readinessHandlerName).Request(reqCtx, "ready")
		return err == nil
	}, 30*time.Second, 250*time.Millisecond, "restate never routed an invocation to the test worker")

	mysqlCfg := containers.MySQL(t)
	database, err := db.New(mysqlCfg.DSN)
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
	}

	ctrlCtx, ctrlCancel := context.WithCancel(ctx)
	t.Cleanup(ctrlCancel)

	go func() {
		require.NoError(t, Run(ctrlCtx, apiConfig))
	}()

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
		ctx:       ctx,
		CtrlURL:   ctrlURL,
		DB:        database,
		Seed:      seeder,
		Secret:    secret,
		AuthToken: authToken,
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

type restateRegistration struct {
	adminURL   string
	registerAs string
}

// register registers the harness's worker as a Restate deployment and returns
// its deployment id so the caller can deregister it on cleanup.
func (r *restateRegistration) register(ctx context.Context) (string, error) {
	registerURL := r.adminURL + "/deployments"
	payload := []byte("{\"uri\": \"" + r.registerAs + "\"}")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, registerURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	requireJSON(req)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmtStatus(resp.StatusCode)
	}

	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	return body.ID, nil
}

// deregister removes the deployment from Restate. Each test registers a fresh
// worker deployment on the shared Restate container; without removing them the
// container collects one dead deployment per test run. force=true drops it even
// while invocations reference it. Best effort: cleanup failures must not fail
// the test.
func (r *restateRegistration) deregister(ctx context.Context, id string) {
	if id == "" {
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, r.adminURL+"/deployments/"+id+"?force=true", nil)
	if err != nil {
		return
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	_ = resp.Body.Close()
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
