package containers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
	restateServer "github.com/restatedev/sdk-go/server"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sys/unix"
)

const (
	restatePort                 = 8080
	restateAdminPort            = 9070
	restateReadinessHandlerName = "ping"
)

var restateWorkerID atomic.Uint64

// RestateConfig holds connection information for the Restate test container.
type RestateConfig struct {
	// IngressURL is the Restate ingress endpoint URL.
	IngressURL string
	// AdminURL is the Restate admin endpoint URL.
	AdminURL string
}

// Restate registers services on the shared Restate test container.
//
// Every registration against the shared container must use Restate. The helper
// takes cross-process leases for all supplied service names, resets their
// invocations and state, registers a temporary worker, and releases the leases
// only after that worker is drained and deregistered. This keeps one container
// for the suite without allowing concurrent Go test binaries to change each
// other's routing.
// Callers must supply every service that the registered handlers can invoke.
func Restate(t *testing.T, services ...restate.ServiceDefinition) RestateConfig {
	t.Helper()
	require.NotEmpty(t, services, "at least one Restate service is required")

	c := startService(t, "restate")
	cfg := RestateConfig{
		IngressURL: fmt.Sprintf("http://%s", c.Addr(t, restatePort)),
		AdminURL:   fmt.Sprintf("http://localhost:%d", c.Port(t, restateAdminPort)),
	}

	serviceNames := sortedServiceNames(services)
	releaseLeases, err := acquireRestateServiceLeases(serviceNames)
	require.NoError(t, err)

	admin := restateAdminClient{
		baseURL: cfg.AdminURL,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
	var worker *httptest.Server
	var deploymentID string
	cleanupServiceNames := serviceNames
	t.Cleanup(func() {
		defer func() {
			if releaseErr := releaseLeases(); releaseErr != nil {
				t.Errorf("release Restate service leases: %v", releaseErr)
			}
		}()
		defer func() {
			if worker != nil {
				worker.Close()
			}
		}()

		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if cleanupErr := admin.drainInvocations(cleanupCtx, cleanupServiceNames); cleanupErr != nil {
			t.Errorf("drain Restate test invocations: %v", cleanupErr)
		}
		if cleanupErr := admin.purgeCompletedInvocations(cleanupCtx, cleanupServiceNames); cleanupErr != nil {
			t.Errorf("purge Restate test invocations: %v", cleanupErr)
		}
		if deploymentID != "" {
			if cleanupErr := admin.deleteDeployment(cleanupCtx, deploymentID); cleanupErr != nil {
				t.Errorf("deregister Restate test worker: %v", cleanupErr)
			}
		}
	})

	resetCtx, resetCancel := context.WithTimeout(t.Context(), 30*time.Second)
	resetErr := admin.resetServices(resetCtx, serviceNames)
	resetCancel()
	require.NoError(t, resetErr)

	readinessServiceName := fmt.Sprintf(
		"unkey.test.RestateReadiness_%d_%d",
		os.Getpid(),
		restateWorkerID.Add(1),
	)
	cleanupServiceNames = append(append([]string{}, serviceNames...), readinessServiceName)
	restateSrv := restateServer.NewRestate()
	for _, service := range services {
		restateSrv.Bind(service)
	}
	restateSrv.Bind(restate.NewObject(readinessServiceName).
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
	deploymentID, err = admin.registerDeployment(
		registerCtx,
		fmt.Sprintf("http://host.docker.internal:%d", workerPort),
	)
	registerCancel()
	require.NoError(t, err)

	readinessCtx, readinessCancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer readinessCancel()
	require.Eventually(t, func() bool {
		requestCtx, requestCancel := context.WithTimeout(readinessCtx, 2*time.Second)
		defer requestCancel()
		_, err := ingress.Object[string, string](
			ingress.NewClient(cfg.IngressURL),
			readinessServiceName,
			"probe",
			restateReadinessHandlerName,
		).Request(requestCtx, "ready")
		return err == nil
	}, 30*time.Second, 250*time.Millisecond, "restate never became ready for keyed invocations")

	return cfg
}

func sortedServiceNames(services []restate.ServiceDefinition) []string {
	unique := make(map[string]struct{}, len(services))
	for _, service := range services {
		unique[service.Name()] = struct{}{}
	}
	names := make([]string, 0, len(unique))
	for name := range unique {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// acquireRestateServiceLeases serializes registrations for overlapping service
// sets across the separate Go test binaries that Rask runs concurrently.
func acquireRestateServiceLeases(serviceNames []string) (func() error, error) {
	files := make([]*os.File, 0, len(serviceNames))
	release := func() error {
		var errs []error
		for i := len(files) - 1; i >= 0; i-- {
			if err := unix.Flock(int(files[i].Fd()), unix.LOCK_UN); err != nil {
				errs = append(errs, err)
			}
			if err := files[i].Close(); err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			return fmt.Errorf("release Restate locks: %v", errs)
		}
		return nil
	}

	for _, serviceName := range serviceNames {
		sum := sha256.Sum256([]byte(composeProjectName() + "\x00" + serviceName))
		lockPath := filepath.Join(
			os.TempDir(),
			"unkey-restate-"+hex.EncodeToString(sum[:8])+".lock",
		)
		file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
		if err != nil {
			_ = release()
			return nil, fmt.Errorf("open Restate lock for %s: %w", serviceName, err)
		}
		files = append(files, file)
		if err := unix.Flock(int(file.Fd()), unix.LOCK_EX); err != nil {
			_ = release()
			return nil, fmt.Errorf("lock Restate service %s: %w", serviceName, err)
		}
	}
	return release, nil
}

type restateAdminClient struct {
	baseURL string
	http    *http.Client
}

type restateInvocationRow struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Target string `json:"target"`
}

type restateStateRow struct {
	ServiceName string `json:"service_name"`
	ServiceKey  string `json:"service_key"`
}

type restateDeployment struct {
	ID       string                     `json:"id"`
	Services []restateDeploymentService `json:"services"`
}

type restateDeploymentService struct {
	Name string `json:"name"`
}

func (a *restateAdminClient) resetServices(ctx context.Context, serviceNames []string) error {
	if err := a.drainInvocations(ctx, serviceNames); err != nil {
		return fmt.Errorf("drain stale invocations: %w", err)
	}
	if err := a.purgeCompletedInvocations(ctx, serviceNames); err != nil {
		return fmt.Errorf("purge stale invocations: %w", err)
	}
	if err := a.deleteDeploymentsForServices(ctx, serviceNames); err != nil {
		return fmt.Errorf("delete stale deployments: %w", err)
	}
	if err := a.clearState(ctx, serviceNames); err != nil {
		return fmt.Errorf("clear stale state: %w", err)
	}
	return nil
}

func (a *restateAdminClient) drainInvocations(ctx context.Context, serviceNames []string) error {
	return a.mutateInvocations(ctx, serviceNames, "status != 'completed'", "kill")
}

func (a *restateAdminClient) purgeCompletedInvocations(ctx context.Context, serviceNames []string) error {
	return a.mutateInvocations(ctx, serviceNames, "status = 'completed'", "purge")
}

func (a *restateAdminClient) mutateInvocations(
	ctx context.Context,
	serviceNames []string,
	statusPredicate string,
	action string,
) error {
	query := fmt.Sprintf(
		"SELECT id, status, target FROM sys_invocation_status WHERE (%s) AND %s LIMIT 1000",
		targetPredicate(serviceNames),
		statusPredicate,
	)
	for {
		rows, err := restateQuery[restateInvocationRow](ctx, a, query)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		for _, row := range rows {
			status, err := a.request(ctx, http.MethodPatch, "/invocations/"+url.PathEscape(row.ID)+"/"+action, nil, nil)
			completedBeforeKill := action == "kill" && status == http.StatusConflict
			if err != nil && status != http.StatusNotFound && !completedBeforeKill {
				return fmt.Errorf("%s invocation %s (%s): %w", action, row.ID, row.Target, err)
			}
		}
		if err := waitForRestateMutation(ctx); err != nil {
			return fmt.Errorf("%s invocations still present: %v: %w", action, rows, err)
		}
	}
}

func (a *restateAdminClient) clearState(ctx context.Context, serviceNames []string) error {
	query := fmt.Sprintf(
		"SELECT DISTINCT service_name, service_key FROM state WHERE service_name IN (%s)",
		serviceList(serviceNames),
	)
	for {
		rows, err := restateQuery[restateStateRow](ctx, a, query)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		for _, row := range rows {
			payload := map[string]any{
				"version":    nil,
				"object_key": row.ServiceKey,
				"new_state":  map[string]string{},
			}
			_, err := a.request(
				ctx,
				http.MethodPost,
				"/services/"+url.PathEscape(row.ServiceName)+"/state",
				payload,
				nil,
			)
			if err != nil {
				return fmt.Errorf("clear state for %s/%s: %w", row.ServiceName, row.ServiceKey, err)
			}
		}
		if err := waitForRestateMutation(ctx); err != nil {
			return fmt.Errorf("state still present: %v: %w", rows, err)
		}
	}
}

func (a *restateAdminClient) deleteDeploymentsForServices(ctx context.Context, serviceNames []string) error {
	for {
		deployments, err := a.deployments(ctx)
		if err != nil {
			return err
		}
		var deleted bool
		for _, deployment := range deployments {
			if !deploymentOverlaps(deployment, serviceNames) {
				continue
			}
			if err := a.deleteDeployment(ctx, deployment.ID); err != nil {
				return err
			}
			deleted = true
		}
		if !deleted {
			return nil
		}
	}
}

func (a *restateAdminClient) deleteDeployment(ctx context.Context, deploymentID string) error {
	status, err := a.request(
		ctx,
		http.MethodDelete,
		"/deployments/"+url.PathEscape(deploymentID)+"?force=true",
		nil,
		nil,
	)
	if err != nil && status != http.StatusNotFound {
		return err
	}
	for {
		deployments, listErr := a.deployments(ctx)
		if listErr != nil {
			return listErr
		}
		found := false
		for _, deployment := range deployments {
			found = found || deployment.ID == deploymentID
		}
		if !found {
			return nil
		}
		if err := waitForRestateMutation(ctx); err != nil {
			return fmt.Errorf("deployment %s still registered: %w", deploymentID, err)
		}
	}
}

func (a *restateAdminClient) deployments(ctx context.Context) ([]restateDeployment, error) {
	var response struct {
		Deployments []restateDeployment `json:"deployments"`
	}
	_, err := a.request(ctx, http.MethodGet, "/deployments", nil, &response)
	return response.Deployments, err
}

func (a *restateAdminClient) registerDeployment(ctx context.Context, uri string) (string, error) {
	var response struct {
		ID string `json:"id"`
	}
	_, err := a.request(ctx, http.MethodPost, "/deployments", map[string]string{"uri": uri}, &response)
	if err != nil {
		return "", err
	}
	if response.ID == "" {
		return "", fmt.Errorf("Restate registration returned an empty deployment id")
	}
	return response.ID, nil
}

func (a *restateAdminClient) request(
	ctx context.Context,
	method string,
	path string,
	payload any,
	response any,
) (int, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return 0, err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, a.baseURL+path, body)
	if err != nil {
		return 0, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := a.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return resp.StatusCode, fmt.Errorf("Restate admin %s %s returned %s: %s", method, path, resp.Status, message)
	}
	if response != nil {
		if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}

func restateQuery[T any](ctx context.Context, admin *restateAdminClient, query string) ([]T, error) {
	var response struct {
		Rows []T `json:"rows"`
	}
	encoded, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, admin.baseURL+"/query", bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := admin.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return nil, fmt.Errorf("Restate query returned %s: %s", resp.Status, message)
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response.Rows, nil
}

func deploymentOverlaps(deployment restateDeployment, serviceNames []string) bool {
	for _, deployedService := range deployment.Services {
		if slices.Contains(serviceNames, deployedService.Name) {
			return true
		}
	}
	return false
}

func targetPredicate(serviceNames []string) string {
	predicates := make([]string, 0, len(serviceNames))
	for _, serviceName := range serviceNames {
		predicates = append(predicates, "target LIKE "+sqlString(serviceName+"/%"))
	}
	return strings.Join(predicates, " OR ")
}

func serviceList(serviceNames []string) string {
	quoted := make([]string, len(serviceNames))
	for i, serviceName := range serviceNames {
		quoted[i] = sqlString(serviceName)
	}
	return strings.Join(quoted, ", ")
}

func sqlString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func waitForRestateMutation(ctx context.Context) error {
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
