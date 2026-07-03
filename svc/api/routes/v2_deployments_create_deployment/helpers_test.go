package handler_test

import (
	"context"
	"database/sql"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_create_deployment"
)

// ctrlCapture records what the handler forwarded to the control plane and lets a
// test inject an error to exercise the ctrl-error mapping.
type ctrlCapture struct {
	called bool
	req    *ctrlv1.CreateDeploymentRequest
	err    error
}

func newRoute(h *testutil.Harness, capture *ctrlCapture) *handler.Handler {
	return &handler.Handler{
		DB: h.DB,
		CtrlClient: &testutil.MockDeploymentClient{
			CreateDeploymentFunc: func(ctx context.Context, req *ctrlv1.CreateDeploymentRequest) (*ctrlv1.CreateDeploymentResponse, error) {
				capture.called = true
				capture.req = req
				if capture.err != nil {
					return nil, capture.err
				}
				return &ctrlv1.CreateDeploymentResponse{DeploymentId: "d_test_generated"}, nil
			},
		},
	}
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + rootKey},
	}
}

// setDeploymentImage records a built container image on a deployment, mimicking
// what ctrl persists after a successful build.
func setDeploymentImage(t *testing.T, h *testutil.Harness, deploymentID, image string) {
	t.Helper()
	err := db.Query.UpdateDeploymentImage(context.Background(), h.DB.RW(), db.UpdateDeploymentImageParams{
		Image:     sql.NullString{String: image, Valid: true},
		UpdatedAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		ID:        deploymentID,
	})
	require.NoError(t, err)
}

// connectRepo attaches a GitHub repository connection to an app so git-sourced
// deployments pass the handler's precondition check.
func connectRepo(t *testing.T, h *testutil.Harness, workspaceID, projectID, appID string) {
	t.Helper()
	err := db.Query.InsertGithubRepoConnection(context.Background(), h.DB.RW(), db.InsertGithubRepoConnectionParams{
		WorkspaceID:        workspaceID,
		ProjectID:          projectID,
		AppID:              appID,
		InstallationID:     12345,
		RepositoryID:       67890,
		RepositoryFullName: "acme/api",
		CreatedAt:          time.Now().UnixMilli(),
		UpdatedAt:          sql.NullInt64{Valid: false},
	})
	require.NoError(t, err)
}
