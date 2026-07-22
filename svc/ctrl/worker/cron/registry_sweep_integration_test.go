package cron_test

import (
	"database/sql"
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

func TestRunRegistrySweep_Integration(t *testing.T) {
	h := harness.New(t)
	h.DepotAPI.SetDepotPageSize(2)

	now := time.Now()
	oldEnough := now.Add(-10 * 24 * time.Hour)
	tooFresh := now.Add(-1 * time.Hour)

	ws := h.Seed.CreateWorkspace(h.Ctx)
	project := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      ws.ID,
		Name:             "test-project",
		Slug:             uid.New("slug"),
		DeleteProtection: false,
	})
	require.NoError(t, h.DB.UpdateProjectDepotID(h.Ctx, db.UpdateProjectDepotIDParams{
		DepotProjectID: sql.NullString{String: "dp_live", Valid: true},
		UpdatedAt:      sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
		ID:             project.ID,
	}))
	app := h.Seed.CreateApp(h.Ctx, seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: ws.ID,
		ProjectID:   project.ID,
		Name:        "default",
		Slug:        "default",
	})
	env := h.Seed.CreateEnvironment(h.Ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New(uid.EnvironmentPrefix),
		WorkspaceID:      ws.ID,
		ProjectID:        project.ID,
		AppID:            app.ID,
		Slug:             "production",
		Description:      "",
		SentinelConfig:   nil,
		DeleteProtection: false,
	})
	live := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		ID:            "",
		WorkspaceID:   ws.ID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: env.ID,
		Status:        db.DeploymentsStatusReady,
		CreatedAt:     oldEnough.UnixMilli(),
		UpdatedAt:     sql.NullInt64{Int64: oldEnough.UnixMilli(), Valid: true},
	})

	liveTag := project.ID + "-" + live.ID
	orphanTag := project.ID + "-d_gone123"
	reusedTag := project.ID + "-d_reused123"
	freshOrphanTag := project.ID + "-d_new456"
	reusedImage := "registry.depot.dev/testrepo:" + reusedTag
	reusedBy := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		ID:            "",
		WorkspaceID:   ws.ID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: env.ID,
		Status:        db.DeploymentsStatusReady,
		CreatedAt:     oldEnough.UnixMilli(),
		UpdatedAt:     sql.NullInt64{Int64: oldEnough.UnixMilli(), Valid: true},
	})
	require.NoError(t, h.DB.UpdateDeploymentImage(h.Ctx, db.UpdateDeploymentImageParams{
		Image:     sql.NullString{String: reusedImage, Valid: true},
		UpdatedAt: sql.NullInt64{Int64: oldEnough.UnixMilli(), Valid: true},
		ID:        reusedBy.ID,
	}))

	// Tag whose deployment exists -> kept, no matter how old.
	h.DepotAPI.SeedImage(depotclient.Image{Tags: []string{liveTag}, PushedAt: oldEnough})
	// Tag whose deployment is gone -> deleted.
	h.DepotAPI.SeedImage(depotclient.Image{Tags: []string{orphanTag}, PushedAt: oldEnough})
	// The encoded deployment is gone, but a rebuild still references this tag.
	h.DepotAPI.SeedImage(depotclient.Image{Tags: []string{reusedTag}, PushedAt: oldEnough})
	// Orphaned but pushed too recently -> kept (row write may be in flight).
	h.DepotAPI.SeedImage(depotclient.Image{Tags: []string{freshOrphanTag}, PushedAt: tooFresh})
	// Hand-pushed tag -> skipped, never deleted.
	h.DepotAPI.SeedImage(depotclient.Image{Tags: []string{"latest"}, PushedAt: oldEnough})

	// Depot project backing an existing Unkey project -> kept.
	h.DepotAPI.SeedProject(depotclient.Project{ID: "dp_live", Name: "builds-test-" + project.ID, CreatedAt: oldEnough})
	// Duplicate name with an unreferenced Depot ID -> deleted.
	h.DepotAPI.SeedProject(depotclient.Project{ID: "dp_duplicate", Name: "builds-test-" + project.ID, CreatedAt: oldEnough})
	// Depot project whose Unkey project is gone -> deleted.
	h.DepotAPI.SeedProject(depotclient.Project{ID: "dp_orphan", Name: "builds-test-proj_gone123", CreatedAt: oldEnough})
	// Orphaned but created too recently -> kept.
	h.DepotAPI.SeedProject(depotclient.Project{ID: "dp_fresh", Name: "builds-test-proj_new456", CreatedAt: tooFresh})
	// Different environment's prefix -> never touched.
	h.DepotAPI.SeedProject(depotclient.Project{ID: "dp_foreign", Name: "builds-prod-proj_gone123", CreatedAt: oldEnough})

	client := hydrav1.NewCronServiceIngressClient(h.Restate, "registry-sweep")
	resp, err := client.RunRegistrySweep().Request(h.Ctx, &hydrav1.RunRegistrySweepRequest{})
	require.NoError(t, err)

	require.Equal(t, int64(1), resp.GetTagsDeleted(), "only the old orphaned tag is deleted")
	require.Equal(t, int64(1), resp.GetTagsSkipped(), "the hand-pushed tag is skipped")
	require.Equal(t, int64(2), resp.GetDepotProjectsDeleted(), "the orphan and unreferenced duplicate are deleted")

	require.Equal(t, []string{orphanTag}, h.DepotAPI.DeletedTags())
	require.ElementsMatch(t, []string{"dp_duplicate", "dp_orphan"}, h.DepotAPI.DeletedProjects())
	require.False(t, h.DepotAPI.HasTag(orphanTag))
	require.True(t, h.DepotAPI.HasTag(liveTag))
	require.True(t, h.DepotAPI.HasTag(reusedTag))
	require.False(t, h.DepotAPI.HasProject("dp_orphan"))
	require.False(t, h.DepotAPI.HasProject("dp_duplicate"))
	require.True(t, h.DepotAPI.HasProject("dp_live"))
}
