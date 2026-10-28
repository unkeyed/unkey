package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/apierrors"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestContextUsesTwoStderrLines(t *testing.T) {
	stdout := captureFile(t, &os.Stdout)
	stderr := captureFile(t, &os.Stderr)
	opts := DeployOptions{
		Project:     "payments",
		App:         "api",
		Environment: "preview-eu",
		Branch:      "feature",
		DockerImage: "ghcr.io/example/api:v1",
	}

	printContext(opts)

	require.Empty(t, stdout())
	require.Equal(t, "Project: payments  App: api  Environment: preview-eu\nImage: ghcr.io/example/api:v1\n", stderr())
}

func TestContextEscapesTerminalControls(t *testing.T) {
	stderr := captureFile(t, &os.Stderr)
	printContext(DeployOptions{
		Project:     "payments\x1b[2J",
		App:         "api\rhidden",
		Environment: "preview-eu\nforged",
		DockerImage: "ghcr.io/example/api:v1\u202e",
	})

	require.Equal(t, "Project: payments\\x1b[2J  App: api\\rhidden  Environment: preview-eu\\nforged\nImage: ghcr.io/example/api:v1\\u202e\n", stderr())
}

func TestCompletionPreservesEnvironmentSlugAndStdout(t *testing.T) {
	stdout := captureFile(t, &os.Stdout)
	stderr := captureFile(t, &os.Stderr)
	deploymentID := uid.New(uid.DeploymentPrefix)

	printCompletionInfo(&components.V2DeployGetDeploymentResponseData{
		ID:        deploymentID,
		Status:    components.V2DeployGetDeploymentResponseDataStatusReady,
		Hostnames: []string{"api.example.com", "localhost:8080"},
	}, "preview-eu")

	require.Empty(t, stderr())
	require.Equal(t, "ID: "+deploymentID+"\nStatus: Ready\nEnvironment: preview-eu\nURL: https://api.example.com\nURL: http://localhost:8080\n", stdout())
}

func TestCompletionWithoutHostnames(t *testing.T) {
	stdout := captureFile(t, &os.Stdout)
	deploymentID := uid.New(uid.DeploymentPrefix)
	printCompletionInfo(&components.V2DeployGetDeploymentResponseData{
		ID:     deploymentID,
		Status: components.V2DeployGetDeploymentResponseDataStatusReady,
	}, "preview-eu")

	require.Equal(t, "ID: "+deploymentID+"\nStatus: Ready\nEnvironment: preview-eu\nURLs: none assigned\n", stdout())
}

func TestCompletionEscapesTerminalControls(t *testing.T) {
	stdout := captureFile(t, &os.Stdout)
	deploymentID := uid.New(uid.DeploymentPrefix) + "\x1b[2J"
	printCompletionInfo(&components.V2DeployGetDeploymentResponseData{
		ID:        deploymentID,
		Status:    components.V2DeployGetDeploymentResponseDataStatusReady,
		Hostnames: []string{"api.example.com\x1b]52;c;clipboard\a", "localhost:8080\rhidden"},
	}, "preview\nforged\u202e")

	require.NotContains(t, stdout(), "\x1b")
	require.NotContains(t, stdout(), "\a")
	require.NotContains(t, stdout(), "\r")
	require.NotContains(t, stdout(), "\u202e")
	require.Contains(t, stdout(), `\x1b[2J`)
	require.Contains(t, stdout(), `https://api.example.com\x1b]52;c;clipboard\a`)
	require.Contains(t, stdout(), `http://localhost:8080\rhidden`)
	require.Contains(t, stdout(), `preview\nforged\u202e`)
}

func TestExecuteDeployOutput(t *testing.T) {
	for _, tt := range []struct {
		name         string
		status       components.V2DeployGetDeploymentResponseDataStatus
		createDenied bool
		message      string
		failure      string
	}{
		{name: "ready", status: components.V2DeployGetDeploymentResponseDataStatusReady},
		{name: "deployment failed", status: components.V2DeployGetDeploymentResponseDataStatusFailed, message: "image pull failed", failure: "deployment failed: image pull failed"},
		{name: "deployment error controls", status: components.V2DeployGetDeploymentResponseDataStatusFailed, message: "image\x1b[2J\r\nforged", failure: `deployment failed: image\x1b[2J\r\nforged`},
		{name: "creation denied", createDenied: true, failure: "failed to create deployment: Permission denied"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stdout := captureFile(t, &os.Stdout)
			stderr := captureFile(t, &os.Stderr)
			t.Setenv("TERM", "xterm-256color")
			deploymentID := uid.New(uid.DeploymentPrefix)
			requestID := uid.New(uid.RequestPrefix)
			requests := make(chan components.V2DeployCreateDeploymentRequestBody, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v2/deploy.createDeployment":
					var request components.V2DeployCreateDeploymentRequestBody
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					requests <- request
					if tt.createDenied {
						w.WriteHeader(http.StatusForbidden)
						if err := json.NewEncoder(w).Encode(apierrors.ForbiddenErrorResponse{
							Meta: components.Meta{RequestID: requestID},
							Error_: components.BaseError{
								Detail: "deployment permission required",
								Status: http.StatusForbidden,
							},
						}); err != nil {
							t.Error(err)
						}
						return
					}
					w.WriteHeader(http.StatusCreated)
					if err := json.NewEncoder(w).Encode(components.V2DeployCreateDeploymentResponseBody{
						Meta: components.Meta{RequestID: requestID},
						Data: components.V2DeployCreateDeploymentResponseData{DeploymentID: deploymentID},
					}); err != nil {
						t.Error(err)
					}
				case "/v2/deploy.getDeployment":
					if err := json.NewEncoder(w).Encode(components.V2DeployGetDeploymentResponseBody{
						Meta: components.Meta{RequestID: requestID},
						Data: components.V2DeployGetDeploymentResponseData{
							ID: deploymentID, Status: tt.status, ErrorMessage: &tt.message,
							Hostnames: []string{"api.example.com"},
						},
					}); err != nil {
						t.Error(err)
					}
				default:
					t.Errorf("unexpected request: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			t.Cleanup(server.Close)
			opts := DeployOptions{
				Project: "payments", App: "api", Environment: "preview-eu", Branch: "feature",
				DockerImage: "ghcr.io/example/api:v1", RootKey: uid.New(uid.KeyPrefix), APIBaseURL: server.URL,
			}
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			t.Cleanup(cancel)

			err := executeDeploy(ctx, opts)

			if tt.failure == "" {
				require.NoError(t, err)
				require.Contains(t, stdout(), "ID: "+deploymentID)
				require.Contains(t, stdout(), "Environment: preview-eu\n")
				require.Contains(t, stdout(), "https://api.example.com")
				require.Equal(t, 1, strings.Count(stderr(), "Waiting for deployment\n"))
				require.Equal(t, 1, strings.Count(stderr(), "Deployment ready.\n"))
				require.Equal(t, 1, strings.Count(stdout()+stderr(), deploymentID))
			} else {
				require.ErrorContains(t, err, tt.failure)
				require.Empty(t, stdout())
				require.NotContains(t, stderr(), tt.failure)
				require.NotContains(t, stderr(), "Deployment failed")
				_, writeErr := fmt.Fprintln(os.Stderr, err)
				require.NoError(t, writeErr)
				require.Equal(t, 1, strings.Count(stderr(), tt.failure))
			}
			if tt.createDenied {
				require.Contains(t, stderr(), "Request ID: "+requestID)
				require.NotContains(t, stderr(), "Deployment created")
			} else {
				require.Equal(t, 1, strings.Count(stderr(), "Deployment created\n"))
			}
			require.Contains(t, stderr(), "Project: payments  App: api  Environment: preview-eu\n")
			require.Equal(t, 1, strings.Count(stderr(), "Creating deployment\n"))
			require.NotContains(t, stdout()+stderr(), "\x1b")
			require.NotContains(t, stdout()+stderr(), "\r")
			require.NotContains(t, stdout()+stderr(), "Deployment Progress")
			require.NotContains(t, stdout()+stderr(), "Deployment Complete")
			require.NotContains(t, stdout()+stderr(), "completed successfully")
			select {
			case request := <-requests:
				require.Equal(t, opts.Project, request.Project)
				require.Equal(t, opts.App, request.App)
				require.Equal(t, opts.Environment, request.EnvironmentSlug)
				require.Equal(t, opts.DockerImage, request.DockerImage)
			default:
				t.Fatal("deployment was not submitted")
			}
		})
	}
}

func captureFile(t *testing.T, target **os.File) func() string {
	t.Helper()
	original := *target
	file, err := os.Create(filepath.Join(t.TempDir(), "output"))
	require.NoError(t, err)
	*target = file
	t.Cleanup(func() {
		*target = original
		require.NoError(t, file.Close())
	})
	return func() string {
		t.Helper()
		data, err := os.ReadFile(file.Name())
		require.NoError(t, err)
		return string(data)
	}
}
