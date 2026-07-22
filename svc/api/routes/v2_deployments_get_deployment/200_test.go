package handler_test

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"testing"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/deployfail"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_get_deployment"
)

func TestGetDeployment(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.read_deployment"},
	})

	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:                    uid.New(uid.DeploymentPrefix),
		WorkspaceID:           setup.Workspace.ID,
		ProjectID:             setup.Project.ID,
		AppID:                 setup.App.ID,
		EnvironmentID:         setup.Environment.ID,
		GitBranch:             "main",
		GitCommitSha:          "9f2c1a7",
		GitCommitMessage:      "add KEBAP endpoint",
		GitCommitAuthorHandle: "octocat",
	})

	req := handler.Request{DeploymentId: dep.ID}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.NotNil(t, res.Body)

	d := res.Body.Data
	require.Equal(t, dep.ID, d.Id)
	require.Equal(t, openapi.DeploymentStatusPending, d.Status)
	require.Equal(t, 8080, d.Runtime.Port)
	require.Equal(t, openapi.SIGTERM, d.Runtime.ShutdownSignal)
	require.Equal(t, openapi.Http1, d.Runtime.UpstreamProtocol)
	require.NotNil(t, d.Runtime.Command)
	require.Nil(t, d.Runtime.Healthcheck)

	// Enriched fields wired from the deployment row + relations query.
	require.Equal(t, setup.Environment.Slug, d.Environment)
	require.Equal(t, setup.App.Slug, d.App)
	require.Equal(t, setup.Project.Slug, d.Project)
	require.False(t, d.IsCurrent, "app has no current deployment pointing here")

	// git-sourced: git set from the seeded commit, docker absent.
	require.NotNil(t, d.Git)
	require.Equal(t, "9f2c1a7", d.Git.CommitSha)
	require.NotNil(t, d.Git.Branch)
	require.Equal(t, "main", *d.Git.Branch)
	require.Nil(t, d.Docker)

	// get-only fields are always present (non-nil), even when empty.
	require.NotNil(t, d.AvailableActions)
	require.NotNil(t, d.Domains)
	require.Empty(t, *d.Domains, "no routes seeded")
	require.Empty(t, d.Regions, "no topology seeded")
	require.Nil(t, d.Error, "pending deployment has no error")

	// Internal fields must never appear in the response body.
	for _, leaked := range []string{"k8s_name", "k8sName", "workspace_id", "workspaceId", "sentinel", "encrypted", "build_id", "buildId", "invocation", "github_deployment", "githubDeployment", "\"pk\""} {
		require.False(t, strings.Contains(res.RawBody, leaked), "response leaked internal field %q: %s", leaked, res.RawBody)
	}
}

// TestGetDeploymentFailure exercises the steps -> failure wiring end to end
// against a real DB, which the mapper unit tests cannot: it seeds a failed
// deployment, records a failed step, and asserts the classified failure.
func TestGetDeploymentFailure(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.read_deployment"},
	})

	recordFailedStep := func(deploymentID string, step db.DeploymentStepsStep, msg string) {
		ctx := context.Background()
		require.NoError(t, db.Query.InsertDeploymentStep(ctx, h.DB.RW(), db.InsertDeploymentStepParams{
			WorkspaceID:   setup.Workspace.ID,
			ProjectID:     setup.Project.ID,
			AppID:         setup.App.ID,
			EnvironmentID: setup.Environment.ID,
			DeploymentID:  deploymentID,
			Step:          step,
			StartedAt:     1,
		}))
		require.NoError(t, db.Query.EndDeploymentStep(ctx, h.DB.RW(), db.EndDeploymentStepParams{
			DeploymentID: deploymentID,
			Step:         step,
			EndedAt:      sql.NullInt64{Valid: true, Int64: 2},
			Error:        sql.NullString{Valid: true, String: msg},
		}))
	}

	newFailedDeployment := func() string {
		return h.CreateDeployment(seed.CreateDeploymentRequest{
			ID:            uid.New(uid.DeploymentPrefix),
			WorkspaceID:   setup.Workspace.ID,
			ProjectID:     setup.Project.ID,
			AppID:         setup.App.ID,
			EnvironmentID: setup.Environment.ID,
			Status:        mysqltype.DeploymentsStatusFailed,
		}).ID
	}

	get := func(id string) openapi.Deployment {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{DeploymentId: id})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		return res.Body.Data
	}

	t.Run("deploying-step failure classified by message", func(t *testing.T) {
		id := newFailedDeployment()
		recordFailedStep(id, db.DeploymentStepsStepDeploying, deployfail.MsgNoSchedulableRegions)

		d := get(id)
		require.Equal(t, openapi.DeploymentStatusFailed, d.Status)
		require.NotNil(t, d.Error)
		require.Equal(t, openapi.DeploymentErrorCodeNoSchedulableRegions, d.Error.Code)
		require.Equal(t, "deploying", d.Error.Step)
		require.Equal(t, deployfail.MsgNoSchedulableRegions, d.Error.Message)
		require.Empty(t, d.AvailableActions, "failed deployment has no legal actions")
	})

	t.Run("build-step failure classified by step", func(t *testing.T) {
		id := newFailedDeployment()
		recordFailedStep(id, db.DeploymentStepsStepBuilding, "some opaque depot build output")

		d := get(id)
		require.NotNil(t, d.Error)
		require.Equal(t, openapi.DeploymentErrorCodeBuildFailed, d.Error.Code)
		require.Equal(t, "building", d.Error.Step)
	})
}

// TestGetDeploymentRegionsAndDomains covers the region-name join and the
// frontline-only domain query end to end, which the mapper unit tests cannot
// because the enrichment is pure SQL.
func TestGetDeploymentRegionsAndDomains(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.read_deployment"},
	})

	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
		Status:        mysqltype.DeploymentsStatusReady,
	})

	var wantRegions []string
	for range 2 {
		regionID := uid.New(uid.RegionPrefix)
		regionName := uid.New(uid.RegionPrefix)
		wantRegions = append(wantRegions, regionName)
		require.NoError(t, db.Query.UpsertRegion(ctx, h.DB.RW(), db.UpsertRegionParams{
			ID: regionID, Name: regionName, Platform: "aws",
		}))
		require.NoError(t, db.Query.InsertDeploymentTopology(ctx, h.DB.RW(), db.InsertDeploymentTopologyParams{
			WorkspaceID:            setup.Workspace.ID,
			DeploymentID:           dep.ID,
			RegionID:               regionID,
			AutoscalingReplicasMin: 1,
			AutoscalingReplicasMax: 1,
			DesiredStatus:          db.DeploymentTopologyDesiredStatusRunning,
			CreatedAt:              1,
		}))
	}

	var wantDomains []string
	for range 2 {
		fqdn := uid.New(uid.DomainPrefix) + ".unkey.app"
		wantDomains = append(wantDomains, fqdn)
		require.NoError(t, db.Query.InsertFrontlineRoute(ctx, h.DB.RW(), db.InsertFrontlineRouteParams{
			ID:                       uid.New(uid.FrontlineRoutePrefix),
			ProjectID:                setup.Project.ID,
			AppID:                    setup.App.ID,
			DeploymentID:             dep.ID,
			EnvironmentID:            setup.Environment.ID,
			FullyQualifiedDomainName: fqdn,
			Sticky:                   db.FrontlineRoutesStickyNone,
			CreatedAt:                1,
		}))
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{DeploymentId: dep.ID})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	d := res.Body.Data
	require.ElementsMatch(t, wantRegions, d.Regions)
	require.NotNil(t, d.Domains)
	require.ElementsMatch(t, wantDomains, *d.Domains)
}

func TestGetDeploymentSpecificEnvironmentPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	rootKey := h.CreateRootKey(setup.Workspace.ID, "environment."+setup.Environment.ID+".read_deployment")

	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})

	req := handler.Request{DeploymentId: dep.ID}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), req)
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, dep.ID, res.Body.Data.Id)
}
