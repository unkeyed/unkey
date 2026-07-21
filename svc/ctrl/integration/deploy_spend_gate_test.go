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

// TestDeploySpendGate_BlocksCreateAndRebuild proves the spend-cap gate rejects
// both entry points that start compute. The gate lives in createAndDeploy (the
// shared path), so this is the regression guard that the ops Rebuild path can't
// resurrect compute the suspension tore down: gating only CreateDeployment would
// let Rebuild slip past.
func TestDeploySpendGate_BlocksCreateAndRebuild(t *testing.T) {
	const bearer = "test-bearer"

	h := New(t)
	ctx := h.Context()

	// Scaffolding: a project/app/env plus a source deployment to rebuild from.
	dep := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       "us-east-1",
		DesiredState: mysqltype.DeploymentsDesiredStateRunning,
	}).Deployment

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
}
