//go:build integration

package integration

import (
	"database/sql"
	"testing"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	deployment "github.com/unkeyed/unkey/svc/ctrl/services/deployment"
)

// TestDeployWorkspaceGate_BlocksCreateAndRebuild proves both entry points that
// start compute share the same plan and spend-cap gate.
func TestDeployWorkspaceGate_BlocksCreateAndRebuild(t *testing.T) {
	const bearer = "test-bearer"

	h := New(t)
	ctx := h.Context()

	// Scaffolding: a project/app/env plus a source deployment to rebuild from.
	dep := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       "us-east-1",
		DesiredState: mysqltype.DeploymentsDesiredStateRunning,
	})
	region, err := h.DB.FindRegionByPlatformAndName(ctx, db.FindRegionByPlatformAndNameParams{
		Platform: "test",
		Name:     "us-east-1",
	})
	require.NoError(t, err)
	require.NoError(t, h.DB.UpsertAppRegionalSettings(ctx, db.UpsertAppRegionalSettingsParams{
		WorkspaceID:   dep.WorkspaceID,
		AppID:         dep.AppID,
		EnvironmentID: dep.EnvironmentID,
		RegionID:      region.ID,
		Replicas:      1,
		CreatedAt:     h.Now(),
		UpdatedAt:     sql.NullInt64{},
	}))
	_, err = h.DB.RW().ExecContext(ctx, "UPDATE regions SET can_schedule = true WHERE id = ?", region.ID)
	require.NoError(t, err)

	// Suspend the workspace as the spend check would when the budget is hit.
	require.NoError(t, h.DB.SetWorkspaceDeploySpendSuspended(ctx, db.SetWorkspaceDeploySpendSuspendedParams{
		Suspended: true,
		UpdatedAt: sql.NullInt64{Int64: h.Now(), Valid: true},
		ID:        dep.WorkspaceID,
	}))

	svc := deployment.New(deployment.Config{Database: h.DB, Bearer: bearer})

	t.Run("CreateDeployment", func(t *testing.T) {
		req := connect.NewRequest(&ctrlv1.CreateDeploymentRequest{
			ProjectId:       dep.ProjectID,
			AppId:           dep.AppID,
			EnvironmentSlug: "production",
			DockerImage:     "nginx:latest",
		})
		req.Header().Set("Authorization", "Bearer "+bearer)

		_, err := svc.CreateDeployment(ctx, req)
		require.Error(t, err)
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
		require.ErrorContains(t, err, "spend cap")
	})

	t.Run("Rebuild", func(t *testing.T) {
		// force=true skips the newer-sibling guard so we test the spend gate in
		// isolation, not the rebuild precondition.
		_, err := svc.Rebuild(ctx, dep.ID, "spend gate test", true)
		require.Error(t, err)
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
		require.ErrorContains(t, err, "spend cap")
	})

	// A cancelled workspace has no plan and is no longer spend-suspended. Both
	// direct create and the ops rebuild path must remain blocked.
	require.NoError(t, h.DB.ClearWorkspaceDeployPlan(ctx, db.ClearWorkspaceDeployPlanParams{
		ID:        dep.WorkspaceID,
		UpdatedAt: sql.NullInt64{Int64: h.Now(), Valid: true},
	}))
	require.NoError(t, h.DB.SetWorkspaceDeploySpendSuspended(ctx, db.SetWorkspaceDeploySpendSuspendedParams{
		Suspended: false,
		UpdatedAt: sql.NullInt64{Int64: h.Now(), Valid: true},
		ID:        dep.WorkspaceID,
	}))

	t.Run("CreateDeployment without plan", func(t *testing.T) {
		req := connect.NewRequest(&ctrlv1.CreateDeploymentRequest{
			ProjectId:       dep.ProjectID,
			AppId:           dep.AppID,
			EnvironmentSlug: "production",
			DockerImage:     "nginx:latest",
		})
		req.Header().Set("Authorization", "Bearer "+bearer)

		_, err := svc.CreateDeployment(ctx, req)
		require.Error(t, err)
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
		require.ErrorContains(t, err, "no active Compute plan")
	})

	t.Run("Rebuild without plan", func(t *testing.T) {
		_, err := svc.Rebuild(ctx, dep.ID, "plan gate test", true)
		require.Error(t, err)
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
		require.ErrorContains(t, err, "no active Compute plan")
	})
}
