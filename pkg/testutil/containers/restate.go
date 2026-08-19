package containers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
	restateServer "github.com/restatedev/sdk-go/server"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

const (
	restatePort                 = 8080
	restateAdminPort            = 9070
	restateReadinessServiceName = "unkey.test.RestateReadiness"
	restateReadinessHandlerName = "ping"

	// keepRestateEnv leaves the Restate container of a failed test running so
	// its invocation journal and state can be inspected through the admin API.
	keepRestateEnv = "UNKEY_TEST_KEEP_RESTATE"
)

// RestateConfig holds connection information for the Restate test container.
type RestateConfig struct {
	// IngressURL is the Restate ingress endpoint URL.
	IngressURL string
	// AdminURL is the Restate admin endpoint URL.
	AdminURL string
}

// Restate starts a Restate server for the calling test and registers services
// on it.
//
// The container belongs to this test alone. Restate identifies services by
// name, and the names come from protobuf packages, so two tests registering
// their own workers on one server would overwrite each other's routing and
// share virtual object state. A private server is the only isolation that
// survives that, and it costs about a second: the alternative, cleaning a
// shared server between registrations, has to serialize every test that
// touches an overlapping service name.
//
// Callers must supply every service that the registered handlers can invoke.
func Restate(t *testing.T, services ...restate.ServiceDefinition) RestateConfig {
	t.Helper()
	require.NotEmpty(t, services, "at least one Restate service is required")

	container, removeContainer := startIsolatedService(t, "restate")
	cfg := RestateConfig{
		IngressURL: fmt.Sprintf("http://%s", container.Addr(t, restatePort)),
		AdminURL:   fmt.Sprintf("http://localhost:%d", container.Port(t, restateAdminPort)),
	}

	var worker *httptest.Server
	t.Cleanup(func() {
		if t.Failed() && os.Getenv(keepRestateEnv) != "" {
			t.Logf("%s is set: keeping Restate for this test at admin %s", keepRestateEnv, cfg.AdminURL)
			return
		}
		// Remove the server before the worker. Restate holds a long-lived
		// request open per running invocation, and httptest.Server.Close waits
		// for in-flight requests, so closing the worker first would block for
		// as long as an invocation keeps running.
		removeContainer()
		closeWorker(t, worker)
	})

	admin := restateAdminClient{
		baseURL: cfg.AdminURL,
		http:    &http.Client{Timeout: 10 * time.Second}, //nolint:exhaustruct // Defaults are sufficient for tests.
	}
	require.Eventually(t, func() bool {
		healthCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()
		return admin.health(healthCtx) == nil
	}, 60*time.Second, 50*time.Millisecond, "restate admin never became healthy")

	restateSrv := restateServer.NewRestate()
	for _, service := range services {
		restateSrv.Bind(service)
	}
	restateSrv.Bind(restate.NewObject(restateReadinessServiceName).
		Handler(restateReadinessHandlerName, restate.NewObjectHandler(
			func(_ restate.ObjectContext, in string) (string, error) { return in, nil })))

	restateHandler, err := restateSrv.Handler()
	require.NoError(t, err)
	workerListener, err := net.Listen("tcp", "0.0.0.0:0") //nolint:gosec // Restate in Docker must reach the test worker.
	require.NoError(t, err)
	worker = httptest.NewUnstartedServer(h2c.NewHandler(restateHandler, &http2.Server{})) //nolint:exhaustruct // Defaults are sufficient for tests.
	require.NoError(t, worker.Listener.Close())
	worker.Listener = workerListener
	worker.Start()

	workerPort := workerListener.Addr().(*net.TCPAddr).Port
	registerCtx, registerCancel := context.WithTimeout(t.Context(), 30*time.Second)
	err = admin.registerDeployment(
		registerCtx,
		fmt.Sprintf("http://host.docker.internal:%d", workerPort),
	)
	registerCancel()
	require.NoError(t, err)

	// Registration only proves Restate reached the worker. A keyed invocation
	// additionally proves the partition processor took leadership, which is
	// what a Send needs to make progress instead of queueing silently.
	readinessCtx, readinessCancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer readinessCancel()
	require.Eventually(t, func() bool {
		requestCtx, requestCancel := context.WithTimeout(readinessCtx, 2*time.Second)
		defer requestCancel()
		_, err := ingress.Object[string, string](
			ingress.NewClient(cfg.IngressURL),
			restateReadinessServiceName,
			"probe",
			restateReadinessHandlerName,
		).Request(requestCtx, "ready")
		return err == nil
	}, 30*time.Second, 100*time.Millisecond, "restate never became ready for keyed invocations")

	return cfg
}

// closeWorker shuts the test worker down without letting a handler that
// ignores cancellation hang the whole test binary in cleanup.
func closeWorker(t *testing.T, worker *httptest.Server) {
	t.Helper()
	if worker == nil {
		return
	}

	closed := make(chan struct{})
	go func() {
		worker.Close()
		close(closed)
	}()

	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	select {
	case <-closed:
	case <-timer.C:
		t.Errorf("Restate test worker did not shut down: a handler is still running after the server was removed")
	}
}

type restateAdminClient struct {
	baseURL string
	http    *http.Client
}

func (a *restateAdminClient) health(ctx context.Context) error {
	return a.request(ctx, http.MethodGet, "/health", nil, nil)
}

func (a *restateAdminClient) registerDeployment(ctx context.Context, uri string) error {
	var response struct {
		ID string `json:"id"`
	}
	if err := a.request(ctx, http.MethodPost, "/deployments", map[string]string{"uri": uri}, &response); err != nil {
		return err
	}
	if response.ID == "" {
		return fmt.Errorf("Restate registration returned an empty deployment id")
	}
	return nil
}

func (a *restateAdminClient) request(
	ctx context.Context,
	method string,
	path string,
	payload any,
	response any,
) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, a.baseURL+path, body)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := a.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return fmt.Errorf("Restate admin %s %s returned %s: %s", method, path, resp.Status, message)
	}
	if response != nil {
		if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
			return err
		}
	}
	return nil
}
