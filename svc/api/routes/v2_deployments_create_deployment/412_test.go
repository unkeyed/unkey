package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_create_deployment"
)

// TestWorkerRejections pins what a caller is told when the worker refuses a
// create. The gates live in the worker and are tested there; this only covers
// the mapping from outcome to status, error type, and whether the worker's
// detail is shown or masked.
func TestWorkerRejections(t *testing.T) {
	type errorResponse struct {
		Error struct {
			Type   string `json:"type"`
			Detail string `json:"detail"`
		} `json:"error"`
	}

	for _, tc := range []struct {
		name       string
		outcome    hydrav1.CreateOutcome
		detail     string
		status     int
		docs       string
		wantDetail string
	}{
		{
			name:       "no compute plan",
			outcome:    hydrav1.CreateOutcome_CREATE_OUTCOME_NO_COMPUTE_PLAN,
			status:     http.StatusPreconditionFailed,
			docs:       "https://unkey.com/docs/errors/unkey/application/precondition_failed",
			wantDetail: deploygate.MsgNoComputePlan,
		},
		{
			name:       "spend suspended",
			outcome:    hydrav1.CreateOutcome_CREATE_OUTCOME_SPEND_SUSPENDED,
			status:     http.StatusPreconditionFailed,
			docs:       "https://unkey.com/docs/errors/unkey/application/precondition_failed",
			wantDetail: deploygate.StartSpendSuspended.Message(),
		},
		{
			name:       "no repo connection",
			outcome:    hydrav1.CreateOutcome_CREATE_OUTCOME_NO_REPO_CONNECTION,
			status:     http.StatusPreconditionFailed,
			docs:       "https://unkey.com/docs/errors/unkey/application/precondition_failed",
			wantDetail: "This app has no GitHub repository connected.",
		},
		{
			name:       "no source image",
			outcome:    hydrav1.CreateOutcome_CREATE_OUTCOME_NO_SOURCE_IMAGE,
			status:     http.StatusPreconditionFailed,
			docs:       "https://unkey.com/docs/errors/unkey/application/precondition_failed",
			wantDetail: "That deployment never finished building, so there is nothing to redeploy.",
		},
		{
			name:       "no source commit",
			outcome:    hydrav1.CreateOutcome_CREATE_OUTCOME_NO_SOURCE_COMMIT,
			status:     http.StatusPreconditionFailed,
			docs:       "https://unkey.com/docs/errors/unkey/application/precondition_failed",
			wantDetail: "That deployment was a Git build but recorded no commit, so there is nothing to rebuild.",
		},
		{
			name:       "no image configured",
			outcome:    hydrav1.CreateOutcome_CREATE_OUTCOME_NO_IMAGE_CONFIGURED,
			status:     http.StatusPreconditionFailed,
			docs:       "https://unkey.com/docs/errors/unkey/application/precondition_failed",
			wantDetail: "This app deploys a container image but none is configured.",
		},
		{
			name:       "no source",
			outcome:    hydrav1.CreateOutcome_CREATE_OUTCOME_NO_SOURCE,
			status:     http.StatusPreconditionFailed,
			docs:       "https://unkey.com/docs/errors/unkey/application/precondition_failed",
			wantDetail: "Nothing to deploy. Pass a git, image, or deployment source",
		},
		{
			name:       "commit not resolved shows the branch and commit",
			outcome:    hydrav1.CreateOutcome_CREATE_OUTCOME_COMMIT_NOT_RESOLVED,
			detail:     `branch "release" or commit "" not found in acme/api`,
			status:     http.StatusPreconditionFailed,
			docs:       "https://unkey.com/docs/errors/unkey/application/precondition_failed",
			wantDetail: `branch "release" or commit "" not found in acme/api`,
		},
		{
			name:       "newer deployment shows the branch",
			outcome:    hydrav1.CreateOutcome_CREATE_OUTCOME_NEWER_DEPLOYMENT_EXISTS,
			detail:     `a newer active deployment exists on branch "main"`,
			status:     http.StatusPreconditionFailed,
			docs:       "https://unkey.com/docs/errors/unkey/application/precondition_failed",
			wantDetail: `a newer active deployment exists on branch "main"`,
		},
		{
			name:       "environment not deployable shows every field",
			outcome:    hydrav1.CreateOutcome_CREATE_OUTCOME_ENVIRONMENT_NOT_DEPLOYABLE,
			detail:     "Port must be between 1 and 65535 (is 0); no schedulable regions are configured",
			status:     http.StatusBadRequest,
			docs:       "https://unkey.com/docs/errors/unkey/application/invalid_environment_settings",
			wantDetail: "Port must be between 1 and 65535 (is 0); no schedulable regions are configured",
		},
		{
			name:       "invalid image shows the parser error",
			outcome:    hydrav1.CreateOutcome_CREATE_OUTCOME_INVALID_IMAGE,
			detail:     `invalid image reference "Acme/Api:v1": repository name must be lowercase`,
			status:     http.StatusBadRequest,
			docs:       "https://unkey.com/docs/errors/unkey/application/invalid_input",
			wantDetail: `invalid image reference "Acme/Api:v1": repository name must be lowercase`,
		},
		{
			name:       "source deployment not found stays masked",
			outcome:    hydrav1.CreateOutcome_CREATE_OUTCOME_SOURCE_DEPLOYMENT_NOT_FOUND,
			detail:     "source deployment d_KEBAP does not belong to app app_KEBAP",
			status:     http.StatusNotFound,
			docs:       "https://unkey.com/docs/errors/unkey/data/deployment_not_found",
			wantDetail: "The project, app, environment, or deployment does not exist.",
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
				Image:       &openapi.DeploymentSourceImage{DockerImage: "nginx:latest"},
			})
			require.Equal(t, tc.status, res.Status, "received: %s", res.RawBody)
			require.Equal(t, tc.docs, res.Body.Error.Type)
			require.Contains(t, res.Body.Error.Detail, tc.wantDetail)
			if tc.status == http.StatusNotFound {
				require.NotContains(t, res.Body.Error.Detail, tc.detail, "a not-found detail must never reach the caller")
			}
		})
	}
}
