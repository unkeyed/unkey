package openapi

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/dockertest"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// Config holds configuration for schemathesis tests
type Config struct {
	// Endpoint is the API endpoint path to test (e.g., "/v2/apis.createApi")
	Endpoint string

	// NumNodes is the number of API nodes to spin up (deprecated, always 1 for subprocess)
	NumNodes int

	// Timeout for the schemathesis run
	Timeout time.Duration

	// Checks specifies which schemathesis checks to run (default: "all")
	Checks string

	// MaxExamples limits the number of test cases per operation
	MaxExamples int

	// Phases specifies which test phases to run (default: "examples,fuzzing")
	// Note: Coverage phase is excluded by default due to schemathesis bug with
	// OpenAPI 3.1 string constraints (minLength/maxLength not respected)
	// See: https://github.com/schemathesis/schemathesis/discussions/1822
	Phases string
}

// DefaultConfig returns a default configuration
func DefaultConfig(endpoint string) Config {
	return Config{
		Endpoint:    endpoint,
		NumNodes:    1,
		Timeout:     5 * time.Minute,
		Checks:      "all",
		MaxExamples: 50,
		// Skip Coverage phase due to schemathesis bug with OpenAPI 3.1
		// where minLength/maxLength constraints are not respected
		Phases: "examples,fuzzing",
	}
}

// RunSchemathesis runs schemathesis tests for a specific endpoint
// This uses the API as a subprocess to break compile-time dependencies
func RunSchemathesis(t *testing.T, endpoint string) {
	RunSchemathesisWithConfig(t, DefaultConfig(endpoint))
}

// RunSchemathesisWithConfig runs schemathesis tests with custom configuration
// This uses the API as a subprocess to break compile-time dependencies
func RunSchemathesisWithConfig(t *testing.T, cfg Config) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// Skip if Docker is not available
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping schemathesis test")
	}

	// Get API binary path from runfiles
	apiBinaryPath := findAPIBinary(t)

	// Setup containers and database
	mysqlCfg := containers.MySQL(t)
	mysqlCfg.DBName = "unkey"
	mysqlDSN := mysqlCfg.FormatDSN()

	clickhouseDSN := containers.ClickHouse(t)
	redisURL := dockertest.Redis(t)
	kafkaBrokers := containers.Kafka(t)

	// Setup database connection for seeding
	database, err := db.New(db.Config{
		Logger:      logging.NewNoop(),
		PrimaryDSN:  mysqlDSN,
		ReadOnlyDSN: "",
	})
	require.NoError(t, err, "Failed to connect to database")
	defer func() {
		_ = database.Close()
	}()

	// Seed the database
	seeder := seed.New(t, database, nil)
	seeder.Seed(ctx)

	// Create root key with full permissions
	rootKey := seeder.CreateRootKey(ctx, seeder.Resources.UserWorkspace.ID,
		// API permissions
		"api.*.create_api",
		"api.*.read_api",
		"api.*.delete_api",
		"api.*.create_key",
		"api.*.read_key",
		"api.*.update_key",
		"api.*.delete_key",
		"api.*.encrypt_key",
		"api.*.decrypt_key",
		"api.*.verify_key",
		// Identity permissions
		"identity.*.create_identity",
		"identity.*.read_identity",
		"identity.*.update_identity",
		"identity.*.delete_identity",
		// Ratelimit permissions
		"ratelimit.*.limit",
		"ratelimit.*.read_override",
		"ratelimit.*.create_override",
		"ratelimit.*.update_override",
		"ratelimit.*.delete_override",
		// RBAC permissions
		"rbac.*.create_permission",
		"rbac.*.read_permission",
		"rbac.*.delete_permission",
		"rbac.*.create_role",
		"rbac.*.read_role",
		"rbac.*.delete_role",
		// Analytics permissions
		"analytics.*.read_verifications",
	)

	// Find an available port for the API server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "Failed to find available port")
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	// Prepare environment for the API subprocess
	env := []string{
		fmt.Sprintf("HTTP_PORT=%d", port),
		fmt.Sprintf("DATABASE_PRIMARY=%s", mysqlDSN),
		fmt.Sprintf("CLICKHOUSE_URL=%s", clickhouseDSN),
		fmt.Sprintf("REDIS_URL=%s", redisURL),
		fmt.Sprintf("KAFKA_BROKERS=%s", strings.Join(kafkaBrokers, ",")),
		"TEST_MODE=true",
		"REGION=test",
		"INSTANCE_ID=subprocess-test",
		"PLATFORM=test",
		"IMAGE=test",
		"DEBUG_CACHE_HEADERS=true",
		// Vault test key from docker-compose
		"VAULT_MASTER_KEYS=Ch9rZWtfMmdqMFBJdVhac1NSa0ZhNE5mOWlLSnBHenFPENTt7an5MRogENt9Si6wms4pQ2XIvqNSIgNpaBenJmXgcInhu6Nfv2U=",
		"CTRL_URL=http://ctrl:7091",
		"CTRL_TOKEN=your-local-dev-key",
	}

	// Start the API subprocess
	cmd := exec.CommandContext(ctx, apiBinaryPath)
	cmd.Env = append(os.Environ(), env...)

	var apiStdout, apiStderr bytes.Buffer
	cmd.Stdout = &apiStdout
	cmd.Stderr = &apiStderr

	err = cmd.Start()
	require.NoError(t, err, "Failed to start API subprocess")

	// Register cleanup
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
		if apiStderr.Len() > 0 {
			t.Logf("API stderr:\n%s", apiStderr.String())
		}
	})

	// Wait for API to be ready
	serverURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	healthURL := fmt.Sprintf("%s/v2/liveness", serverURL)
	waitForHealthy(t, healthURL, 30*time.Second)

	t.Logf("API subprocess started on %s", serverURL)

	// Write embedded OpenAPI spec to temp file for Docker to mount
	specPath := writeOpenAPISpec(t)

	// Build and run schemathesis
	parsedURL, err := url.Parse(serverURL)
	require.NoError(t, err, "Failed to parse server URL")

	args := buildSchemathesisArgs(cfg, parsedURL, specPath, rootKey)

	t.Logf("Running schemathesis for endpoint: %s", cfg.Endpoint)
	t.Logf("Server URL: %s", serverURL)
	t.Logf("Command: docker %s", strings.Join(args, " "))

	// Run schemathesis
	schemathesisCmd := exec.CommandContext(ctx, "docker", args...)
	var stdout, stderr bytes.Buffer
	schemathesisCmd.Stdout = &stdout
	schemathesisCmd.Stderr = &stderr

	err = schemathesisCmd.Run()
	if err != nil {
		t.Logf("Schemathesis stdout:\n%s", stdout.String())
		t.Logf("Schemathesis stderr:\n%s", stderr.String())
		t.Fatalf("Schemathesis failed for endpoint %s: %v", cfg.Endpoint, err)
	}

	t.Logf("Schemathesis passed for endpoint: %s", cfg.Endpoint)
	if stdout.Len() > 0 {
		t.Logf("Output:\n%s", stdout.String())
	}
}

// findAPIBinary locates the API binary from Bazel runfiles or a fallback path
func findAPIBinary(t *testing.T) string {
	// Try Bazel runfiles first
	runfilesDir := os.Getenv("RUNFILES_DIR")
	if runfilesDir != "" {
		// Standard Bazel runfiles path
		candidates := []string{
			filepath.Join(runfilesDir, "_main/svc/api/cmd/api_/api"),
			filepath.Join(runfilesDir, "unkey/svc/api/cmd/api_/api"),
			filepath.Join(runfilesDir, "_main/svc/api/cmd/api"),
			filepath.Join(runfilesDir, "unkey/svc/api/cmd/api"),
		}
		for _, path := range candidates {
			if _, err := os.Stat(path); err == nil {
				t.Logf("Found API binary at: %s", path)
				return path
			}
		}
	}

	// Try TEST_SRCDIR (another Bazel env var)
	testSrcDir := os.Getenv("TEST_SRCDIR")
	if testSrcDir != "" {
		candidates := []string{
			filepath.Join(testSrcDir, "_main/svc/api/cmd/api_/api"),
			filepath.Join(testSrcDir, "unkey/svc/api/cmd/api_/api"),
		}
		for _, path := range candidates {
			if _, err := os.Stat(path); err == nil {
				t.Logf("Found API binary at: %s", path)
				return path
			}
		}
	}

	// Try relative path from workspace root (for local development)
	workspaceDir := os.Getenv("BUILD_WORKSPACE_DIRECTORY")
	if workspaceDir != "" {
		path := filepath.Join(workspaceDir, "bazel-bin/svc/api/cmd/api_/api")
		if _, err := os.Stat(path); err == nil {
			t.Logf("Found API binary at: %s", path)
			return path
		}
	}

	// Try common Bazel output paths
	commonPaths := []string{
		"bazel-bin/svc/api/cmd/api_/api",
		"../../../bazel-bin/svc/api/cmd/api_/api",
		"../../../../bazel-bin/svc/api/cmd/api_/api",
	}
	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			absPath, _ := filepath.Abs(path)
			t.Logf("Found API binary at: %s", absPath)
			return absPath
		}
	}

	t.Fatal("Could not find API binary. Make sure to build it with: bazel build //svc/api/cmd:api")
	return ""
}

// waitForHealthy waits for the API health endpoint to respond
func waitForHealthy(t *testing.T, healthURL string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := client.Get(healthURL)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("API server did not become healthy within %v", timeout)
}

// buildSchemathesisArgs constructs the Docker command arguments for schemathesis
func buildSchemathesisArgs(cfg Config, serverURL *url.URL, specPath, rootKey string) []string {
	// Get the host for Docker networking
	dockerHost := getDockerHost(serverURL)

	args := []string{
		"run", "--rm",
		// Mount the spec file
		"-v", fmt.Sprintf("%s:/specs/openapi.yaml:ro", specPath),
	}

	// On Linux, use --network=host for direct access
	// On macOS/Windows, Docker Desktop handles host.docker.internal automatically
	if runtime.GOOS == "linux" {
		args = append(args, "--network", "host")
	}

	args = append(args,
		// Schemathesis image
		"schemathesis/schemathesis:stable",
		// Run command
		"run",
		// Spec file
		"/specs/openapi.yaml",
		// Base URL
		"--url", fmt.Sprintf("http://%s", dockerHost),
		// Authorization header
		"--header", fmt.Sprintf("Authorization: Bearer %s", rootKey),
		// Filter to specific endpoint
		"--include-path", cfg.Endpoint,
		// Checks to run
		"--checks", cfg.Checks,
		// Max examples for speed
		"--max-examples", fmt.Sprintf("%d", cfg.MaxExamples),
		// Test phases to run (skip Coverage due to OpenAPI 3.1 bug)
		"--phases", cfg.Phases,
	)

	return args
}

// getDockerHost returns the appropriate host address for Docker to reach the API server
func getDockerHost(serverURL *url.URL) string {
	host := serverURL.Host

	// On macOS/Windows with Docker Desktop, use host.docker.internal
	// On Linux with --network=host, use the actual host
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		// Extract port from the host
		port := serverURL.Port()
		if port == "" {
			port = "80"
		}
		// Always use host.docker.internal on macOS/Windows
		return "host.docker.internal:" + port
	}

	return host
}

// writeOpenAPISpec writes the embedded OpenAPI spec to a temp file and returns the path
func writeOpenAPISpec(t *testing.T) string {
	// Create a temp directory that will be cleaned up after the test
	tempDir := t.TempDir()
	specPath := filepath.Join(tempDir, "openapi-generated.yaml")

	err := os.WriteFile(specPath, openapi.Spec, 0o644)
	require.NoError(t, err, "Failed to write OpenAPI spec to temp file")

	return specPath
}

// isDockerAvailable checks if Docker is available
func isDockerAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "version")
	return cmd.Run() == nil
}

// TestEndpoint represents a single endpoint to test
type TestEndpoint struct {
	Path        string // e.g., "/v2/apis.createApi"
	Method      string // e.g., "POST"
	OperationID string // e.g., "createApi"
	PackageName string // e.g., "v2_apis_createApi"
}

// GetAllEndpoints returns all testable endpoints from the OpenAPI spec
func GetAllEndpoints() []TestEndpoint {
	return []TestEndpoint{
		{Path: "/v2/analytics.getVerifications", Method: "POST", OperationID: "analytics.getVerifications", PackageName: "analytics_getVerifications"},
		{Path: "/v2/apis.createApi", Method: "POST", OperationID: "apis.createApi", PackageName: "apis_createApi"},
		{Path: "/v2/apis.deleteApi", Method: "POST", OperationID: "apis.deleteApi", PackageName: "apis_deleteApi"},
		{Path: "/v2/apis.getApi", Method: "POST", OperationID: "apis.getApi", PackageName: "apis_getApi"},
		{Path: "/v2/apis.listKeys", Method: "POST", OperationID: "apis.listKeys", PackageName: "apis_listKeys"},
		{Path: "/v2/deploy.createDeployment", Method: "POST", OperationID: "deploy.createDeployment", PackageName: "deploy_createDeployment"},
		{Path: "/v2/deploy.getDeployment", Method: "POST", OperationID: "deploy.getDeployment", PackageName: "deploy_getDeployment"},
		{Path: "/v2/identities.createIdentity", Method: "POST", OperationID: "identities.createIdentity", PackageName: "identities_createIdentity"},
		{Path: "/v2/identities.deleteIdentity", Method: "POST", OperationID: "identities.deleteIdentity", PackageName: "identities_deleteIdentity"},
		{Path: "/v2/identities.getIdentity", Method: "POST", OperationID: "identities.getIdentity", PackageName: "identities_getIdentity"},
		{Path: "/v2/identities.listIdentities", Method: "POST", OperationID: "identities.listIdentities", PackageName: "identities_listIdentities"},
		{Path: "/v2/identities.updateIdentity", Method: "POST", OperationID: "identities.updateIdentity", PackageName: "identities_updateIdentity"},
		{Path: "/v2/keys.addPermissions", Method: "POST", OperationID: "keys.addPermissions", PackageName: "keys_addPermissions"},
		{Path: "/v2/keys.addRoles", Method: "POST", OperationID: "keys.addRoles", PackageName: "keys_addRoles"},
		{Path: "/v2/keys.createKey", Method: "POST", OperationID: "keys.createKey", PackageName: "keys_createKey"},
		{Path: "/v2/keys.deleteKey", Method: "POST", OperationID: "keys.deleteKey", PackageName: "keys_deleteKey"},
		{Path: "/v2/keys.getKey", Method: "POST", OperationID: "keys.getKey", PackageName: "keys_getKey"},
		{Path: "/v2/keys.migrateKeys", Method: "POST", OperationID: "keys.migrateKeys", PackageName: "keys_migrateKeys"},
		{Path: "/v2/keys.removePermissions", Method: "POST", OperationID: "keys.removePermissions", PackageName: "keys_removePermissions"},
		{Path: "/v2/keys.removeRoles", Method: "POST", OperationID: "keys.removeRoles", PackageName: "keys_removeRoles"},
		{Path: "/v2/keys.rerollKey", Method: "POST", OperationID: "keys.rerollKey", PackageName: "keys_rerollKey"},
		{Path: "/v2/keys.setPermissions", Method: "POST", OperationID: "keys.setPermissions", PackageName: "keys_setPermissions"},
		{Path: "/v2/keys.setRoles", Method: "POST", OperationID: "keys.setRoles", PackageName: "keys_setRoles"},
		{Path: "/v2/keys.updateCredits", Method: "POST", OperationID: "keys.updateCredits", PackageName: "keys_updateCredits"},
		{Path: "/v2/keys.updateKey", Method: "POST", OperationID: "keys.updateKey", PackageName: "keys_updateKey"},
		{Path: "/v2/keys.verifyKey", Method: "POST", OperationID: "keys.verifyKey", PackageName: "keys_verifyKey"},
		{Path: "/v2/keys.whoami", Method: "POST", OperationID: "keys.whoami", PackageName: "keys_whoami"},
		{Path: "/v2/liveness", Method: "GET", OperationID: "liveness", PackageName: "liveness"},
		{Path: "/v2/permissions.createPermission", Method: "POST", OperationID: "permissions.createPermission", PackageName: "permissions_createPermission"},
		{Path: "/v2/permissions.createRole", Method: "POST", OperationID: "permissions.createRole", PackageName: "permissions_createRole"},
		{Path: "/v2/permissions.deletePermission", Method: "POST", OperationID: "permissions.deletePermission", PackageName: "permissions_deletePermission"},
		{Path: "/v2/permissions.deleteRole", Method: "POST", OperationID: "permissions.deleteRole", PackageName: "permissions_deleteRole"},
		{Path: "/v2/permissions.getPermission", Method: "POST", OperationID: "permissions.getPermission", PackageName: "permissions_getPermission"},
		{Path: "/v2/permissions.getRole", Method: "POST", OperationID: "permissions.getRole", PackageName: "permissions_getRole"},
		{Path: "/v2/permissions.listPermissions", Method: "POST", OperationID: "permissions.listPermissions", PackageName: "permissions_listPermissions"},
		{Path: "/v2/permissions.listRoles", Method: "POST", OperationID: "permissions.listRoles", PackageName: "permissions_listRoles"},
		{Path: "/v2/ratelimit.deleteOverride", Method: "POST", OperationID: "ratelimit.deleteOverride", PackageName: "ratelimit_deleteOverride"},
		{Path: "/v2/ratelimit.getOverride", Method: "POST", OperationID: "ratelimit.getOverride", PackageName: "ratelimit_getOverride"},
		{Path: "/v2/ratelimit.limit", Method: "POST", OperationID: "ratelimit.limit", PackageName: "ratelimit_limit"},
		{Path: "/v2/ratelimit.listOverrides", Method: "POST", OperationID: "ratelimit.listOverrides", PackageName: "ratelimit_listOverrides"},
		{Path: "/v2/ratelimit.multiLimit", Method: "POST", OperationID: "ratelimit.multiLimit", PackageName: "ratelimit_multiLimit"},
		{Path: "/v2/ratelimit.setOverride", Method: "POST", OperationID: "ratelimit.setOverride", PackageName: "ratelimit_setOverride"},
	}
}
