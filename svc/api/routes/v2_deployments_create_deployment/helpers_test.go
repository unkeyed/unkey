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
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_create_deployment"
)

// imageRequest builds a create-deployment request for the image source.
func imageRequest(t *testing.T, project, app, env, dockerImage string) handler.Request {
	t.Helper()
	return handler.Request{
		Project:     project,
		App:         app,
		Environment: env,
		Image:       &openapi.DeploymentSourceImage{DockerImage: dockerImage},
	}
}

// gitRequest builds a create-deployment request for the git source. Callers set
// branch, commitSha, and repository on the passed value.
func gitRequest(t *testing.T, project, app, env string, git openapi.DeploymentSourceGit) handler.Request {
	t.Helper()
	return handler.Request{
		Project:     project,
		App:         app,
		Environment: env,
		Git:         &git,
	}
}

// deploymentRequest builds a create-deployment request for the deployment
// (redeploy) source.
func deploymentRequest(t *testing.T, project, app, env, deploymentID string) handler.Request {
	t.Helper()
	return handler.Request{
		Project:     project,
		App:         app,
		Environment: env,
		Deployment:  &openapi.DeploymentSourceDeployment{DeploymentId: deploymentID},
	}
}

// ctrlCapture records what the handler forwarded to the control plane and lets a
// test inject an error to exercise the ctrl-error mapping.
type ctrlCapture struct {
	called bool
	req    *ctrlv1.CreateDeploymentRequest
	resp   *ctrlv1.CreateDeploymentResponse
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
				if capture.resp != nil {
					return capture.resp, nil
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

// seedDeployableRegion attaches a schedulable region to the setup's environment
// so it clears the create handler's deployability pre-flight and reaches ctrl.
// The seeder gives an environment sane runtime settings but no region.
func seedDeployableRegion(t *testing.T, h *testutil.Harness, setup testutil.DeploymentTestSetup) {
	t.Helper()
	ctx := context.Background()
	regionID := uid.New(uid.RegionPrefix)
	require.NoError(t, db.Query.UpsertRegion(ctx, h.DB.RW(), db.UpsertRegionParams{
		ID:       regionID,
		Name:     uid.New(uid.RegionPrefix),
		Platform: "test",
	}))
	require.NoError(t, db.Query.UpsertAppRegionalSettings(ctx, h.DB.RW(), db.UpsertAppRegionalSettingsParams{
		WorkspaceID:   setup.Workspace.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
		RegionID:      regionID,
		Replicas:      1,
		CreatedAt:     time.Now().UnixMilli(),
		UpdatedAt:     sql.NullInt64{Valid: false},
	}))
}

// zeroRuntimeSettings overwrites an environment's runtime settings with the
// undeployable zero defaults a freshly seeded environment used to carry, so the
// create handler's pre-flight rejects the port/cpu/memory bounds.
func zeroRuntimeSettings(t *testing.T, h *testutil.Harness, setup testutil.DeploymentTestSetup) {
	t.Helper()
	err := db.Query.UpsertAppRuntimeSettings(context.Background(), h.DB.RW(), db.UpsertAppRuntimeSettingsParams{
		WorkspaceID:      setup.Workspace.ID,
		AppID:            setup.App.ID,
		EnvironmentID:    setup.Environment.ID,
		Port:             0,
		CpuMillicores:    0,
		MemoryMib:        0,
		StorageMib:       0,
		Command:          nil,
		Healthcheck:      dbtype.NullHealthcheck{Valid: false},
		ShutdownSignal:   db.AppRuntimeSettingsShutdownSignalSIGTERM,
		UpstreamProtocol: db.AppRuntimeSettingsUpstreamProtocolHttp1,
		SentinelConfig:   []byte("{}"),
		CreatedAt:        time.Now().UnixMilli(),
		UpdatedAt:        sql.NullInt64{Valid: false},
		OpenapiSpecPath:  sql.NullString{Valid: false},
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
