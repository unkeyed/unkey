package project_test

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/depotclient"
)

// TestDeleteRemovesDepotProjectBeforeDatabaseRow verifies that project teardown
// also reclaims the external build cache referenced only by the project row.
func TestDeleteRemovesDepotProjectBeforeDatabaseRow(t *testing.T) {
	h := harness.New(t)
	workspace := h.Seed.CreateWorkspace(h.Ctx)
	project := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      workspace.ID,
		Name:             "project cleanup",
		Slug:             uid.New("slug"),
		DeleteProtection: false,
	})
	depotProjectID := uid.New("depot")
	h.DepotAPI.SeedProject(depotclient.Project{
		ID:        depotProjectID,
		Name:      "builds-test-" + project.ID,
		CreatedAt: time.Now().Add(-time.Hour),
	})
	require.NoError(t, h.DB.UpdateProjectDepotID(h.Ctx, db.UpdateProjectDepotIDParams{
		DepotProjectID: sql.NullString{String: depotProjectID, Valid: true},
		UpdatedAt:      sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		ID:             project.ID,
	}))

	client := hydrav1.NewProjectServiceIngressClient(h.Restate, project.ID)
	_, err := client.Delete().Request(h.Ctx, &hydrav1.DeleteProjectRequest{})
	require.NoError(t, err)
	require.Equal(t, []string{depotProjectID}, h.DepotAPI.DeletedProjects())
	require.False(t, h.DepotAPI.HasProject(depotProjectID))

	_, err = h.DB.FindProjectById(h.Ctx, project.ID)
	require.True(t, db.IsNotFound(err), "project row should be deleted, got %v", err)
}

// TestDeletePreservesProjectWhenDepotDeletionFails guarantees the only stored
// Depot project reference is not lost when the external deletion fails.
func TestDeletePreservesProjectWhenDepotDeletionFails(t *testing.T) {
	h := harness.New(t)
	workspace := h.Seed.CreateWorkspace(h.Ctx)
	project := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      workspace.ID,
		Name:             "failed project cleanup",
		Slug:             uid.New("slug"),
		DeleteProtection: false,
	})
	depotProjectID := uid.New("depot")
	h.DepotAPI.SeedProject(depotclient.Project{
		ID:        depotProjectID,
		Name:      "builds-test-" + project.ID,
		CreatedAt: time.Now().Add(-time.Hour),
	})
	require.NoError(t, h.DB.UpdateProjectDepotID(h.Ctx, db.UpdateProjectDepotIDParams{
		DepotProjectID: sql.NullString{String: depotProjectID, Valid: true},
		UpdatedAt:      sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		ID:             project.ID,
	}))
	h.DepotAPI.SetDeleteProjectError(errors.New("depot unavailable"))

	client := hydrav1.NewProjectServiceIngressClient(h.Restate, project.ID)
	_, err := client.Delete().Request(h.Ctx, &hydrav1.DeleteProjectRequest{})
	require.Error(t, err)
	require.True(t, h.DepotAPI.HasProject(depotProjectID))

	_, err = h.DB.FindProjectById(h.Ctx, project.ID)
	require.NoError(t, err, "project row must retain the external project reference")
}
