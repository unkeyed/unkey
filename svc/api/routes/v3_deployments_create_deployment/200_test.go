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
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v3_deployments_create_deployment"
)

func TestCreateOCIDeployment(t *testing.T) {
	h := testutil.NewHarness(t)
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	seedDeployableRegion(t, h, setup)

	var captured *ctrlv1.CreateDeploymentRequest
	route := &handler.Handler{
		DB: h.DB,
		CtrlClient: &testutil.MockDeploymentClient{
			CreateDeploymentFunc: func(_ context.Context, req *ctrlv1.CreateDeploymentRequest) (*ctrlv1.CreateDeploymentResponse, error) {
				captured = req
				return &ctrlv1.CreateDeploymentResponse{DeploymentId: "d_test_generated"}, nil
			},
		},
	}
	h.Register(route)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		Project:     setup.Project.Slug,
		App:         setup.App.Slug,
		Environment: setup.Environment.Slug,
		Oci:         &openapi.DeploymentSourceOCI{Image: "nginx:latest"},
	})
	require.Equal(t, http.StatusCreated, res.Status, "received: %s", res.RawBody)
	require.Equal(t, "d_test_generated", res.Body.Data.DeploymentId)
	require.NotNil(t, captured)
	require.Equal(t, "nginx:latest", captured.GetOciImage())
}

func TestCreateDeploymentWithAppDefault(t *testing.T) {
	h := testutil.NewHarness(t)
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	seedDeployableRegion(t, h, setup)

	var captured *ctrlv1.CreateDeploymentRequest
	route := &handler.Handler{
		DB: h.DB,
		CtrlClient: &testutil.MockDeploymentClient{
			CreateDeploymentFunc: func(_ context.Context, req *ctrlv1.CreateDeploymentRequest) (*ctrlv1.CreateDeploymentResponse, error) {
				captured = req
				return &ctrlv1.CreateDeploymentResponse{DeploymentId: "d_test_generated"}, nil
			},
		},
	}
	h.Register(route)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		Project:     setup.Project.Slug,
		App:         setup.App.Slug,
		Environment: setup.Environment.Slug,
	})
	require.Equal(t, http.StatusCreated, res.Status, "received: %s", res.RawBody)
	require.NotNil(t, captured)
	require.Empty(t, captured.GetOciImage())
	require.Nil(t, captured.GetGitCommit())
}

func TestCreateGitDeployment(t *testing.T) {
	h := testutil.NewHarness(t)
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	seedDeployableRegion(t, h, setup)
	connectRepo(t, h, setup)

	var captured *ctrlv1.CreateDeploymentRequest
	route := &handler.Handler{
		DB: h.DB,
		CtrlClient: &testutil.MockDeploymentClient{
			CreateDeploymentFunc: func(_ context.Context, req *ctrlv1.CreateDeploymentRequest) (*ctrlv1.CreateDeploymentResponse, error) {
				captured = req
				return &ctrlv1.CreateDeploymentResponse{DeploymentId: "d_test_generated"}, nil
			},
		},
	}
	h.Register(route)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		Project:     setup.Project.Slug,
		App:         setup.App.Slug,
		Environment: setup.Environment.Slug,
		Git:         &openapi.DeploymentSourceGit{Branch: ptr.P("main")},
	})
	require.Equal(t, http.StatusCreated, res.Status, "received: %s", res.RawBody)
	require.NotNil(t, captured)
	require.NotNil(t, captured.GetGitCommit())
	require.Equal(t, "main", captured.GetGitCommit().GetBranch())
}

func TestRedeployOCIUsesResolvedImage(t *testing.T) {
	h := testutil.NewHarness(t)
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	seedDeployableRegion(t, h, setup)

	deployment := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})
	requested := "ghcr.io/acme/api:stable"
	resolved := "ghcr.io/acme/api@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	_, err := h.DB.RW().ExecContext(context.Background(), `
		UPDATE deployments
		SET source = ?, image_requested = ?, image_resolved = ?, image = ?
		WHERE id = ?
	`, db.DeploymentsSourceOci, requested, resolved, resolved, deployment.ID)
	require.NoError(t, err)

	var captured *ctrlv1.CreateDeploymentRequest
	route := &handler.Handler{
		DB: h.DB,
		CtrlClient: &testutil.MockDeploymentClient{
			CreateDeploymentFunc: func(_ context.Context, req *ctrlv1.CreateDeploymentRequest) (*ctrlv1.CreateDeploymentResponse, error) {
				captured = req
				return &ctrlv1.CreateDeploymentResponse{DeploymentId: "d_test_generated"}, nil
			},
		},
	}
	h.Register(route)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		Project:     setup.Project.Slug,
		App:         setup.App.Slug,
		Environment: setup.Environment.Slug,
		Deployment:  &openapi.DeploymentSourceDeployment{DeploymentId: deployment.ID},
	})
	require.Equal(t, http.StatusCreated, res.Status, "received: %s", res.RawBody)
	require.NotNil(t, captured)
	require.Equal(t, resolved, captured.GetOciImage())
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + rootKey},
	}
}

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

func connectRepo(t *testing.T, h *testutil.Harness, setup testutil.DeploymentTestSetup) {
	t.Helper()
	require.NoError(t, db.Query.InsertGithubRepoConnection(context.Background(), h.DB.RW(), db.InsertGithubRepoConnectionParams{
		WorkspaceID:        setup.Workspace.ID,
		ProjectID:          setup.Project.ID,
		AppID:              setup.App.ID,
		InstallationID:     12345,
		RepositoryID:       67890,
		RepositoryFullName: "acme/api",
		DefaultBranch:      sql.NullString{Valid: true, String: "main"},
		CreatedAt:          time.Now().UnixMilli(),
		UpdatedAt:          sql.NullInt64{Valid: false},
	}))
}
