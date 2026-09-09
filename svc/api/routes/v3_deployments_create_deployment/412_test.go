package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v3_deployments_create_deployment"
)

// TestWorkerRejections pins that a refusal from the worker never becomes a 201
// on this route, and that a detail about the caller's own input reaches them.
func TestWorkerRejections(t *testing.T) {
	type errorResponse struct {
		Error struct {
			Type   string `json:"type"`
			Detail string `json:"detail"`
		} `json:"error"`
	}

	for _, tc := range []struct {
		name    string
		outcome hydrav1.CreateOutcome
		detail  string
		status  int
		docs    string
	}{
		{
			name:    "environment not deployable",
			outcome: hydrav1.CreateOutcome_CREATE_OUTCOME_ENVIRONMENT_NOT_DEPLOYABLE,
			detail:  "Port must be between 1 and 65535 (is 0)",
			status:  http.StatusBadRequest,
			docs:    "https://unkey.com/docs/errors/unkey/application/invalid_environment_settings",
		},
		{
			name:    "invalid image",
			outcome: hydrav1.CreateOutcome_CREATE_OUTCOME_INVALID_IMAGE,
			detail:  `invalid image reference "Acme/Api:v1": repository name must be lowercase`,
			status:  http.StatusBadRequest,
			docs:    "https://unkey.com/docs/errors/unkey/application/invalid_input",
		},
		{
			name:    "commit not resolved",
			outcome: hydrav1.CreateOutcome_CREATE_OUTCOME_COMMIT_NOT_RESOLVED,
			detail:  `branch "release" or commit "" not found in acme/api`,
			status:  http.StatusPreconditionFailed,
			docs:    "https://unkey.com/docs/errors/unkey/application/precondition_failed",
		},
		{
			name:    "newer deployment exists",
			outcome: hydrav1.CreateOutcome_CREATE_OUTCOME_NEWER_DEPLOYMENT_EXISTS,
			detail:  `a newer active deployment exists on branch "main"`,
			status:  http.StatusPreconditionFailed,
			docs:    "https://unkey.com/docs/errors/unkey/application/precondition_failed",
		},
		{
			name:    "source deployment not found stays masked",
			outcome: hydrav1.CreateOutcome_CREATE_OUTCOME_SOURCE_DEPLOYMENT_NOT_FOUND,
			detail:  "source deployment d_KEBAP does not belong to app app_KEBAP",
			status:  http.StatusNotFound,
			docs:    "https://unkey.com/docs/errors/unkey/data/deployment_not_found",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := testutil.NewHarness(t)
			setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
				Permissions: []string{"environment.*.create_deployment"},
			})
			route := &handler.Handler{DB: h.DB, Restate: testutil.RejectingDeployRestate(t, tc.outcome, tc.detail)}
			h.Register(route)

			res := testutil.CallRoute[handler.Request, errorResponse](h, route, authHeaders(setup.RootKey), handler.Request{
				Project:     setup.Project.Slug,
				App:         setup.App.Slug,
				Environment: setup.Environment.Slug,
				Oci:         &openapi.DeploymentSourceOCI{Image: "nginx:latest"},
			})
			require.Equal(t, tc.status, res.Status, "received: %s", res.RawBody)
			require.Equal(t, tc.docs, res.Body.Error.Type)
			if tc.status == http.StatusNotFound {
				require.NotContains(t, res.Body.Error.Detail, tc.detail, "a not-found detail must never reach the caller")
				return
			}
			require.Contains(t, res.Body.Error.Detail, tc.detail)
		})
	}
}
