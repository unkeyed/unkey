package cron_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
)

func TestRunPermanentDelete_Integration(t *testing.T) {
	fixed := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	tc := clock.NewTestClock(fixed)
	h := harness.New(t, harness.WithClock(tc))

	t.Run("triggers cascade for project past its delete_permanently_at", func(t *testing.T) {
		ws := h.Seed.CreateWorkspace(h.Ctx)
		proj := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
			ID:          uid.New(uid.ProjectPrefix),
			WorkspaceID: ws.ID,
			Name:        "due-project",
			Slug:        uid.New("slug"),
		})

		// One hour past the cutoff.
		stampDueProject(t, h, ws.ID, proj.ID, fixed.Add(-1*time.Hour).UnixMilli())

		resp, err := callRunPermanentDelete(h, "due")
		require.NoError(t, err)
		require.GreaterOrEqual(t, resp.GetTotalTriggered(), int32(1))

		// Sweeps slice is unordered; at least one entry must be the
		// project sweep with a non-zero count.
		var found bool
		for _, s := range resp.GetSweeps() {
			if s.GetResource() == "project" && s.GetTriggered() >= 1 {
				found = true
				break
			}
		}
		require.True(t, found, "expected a project sweep with triggered >= 1")

		// Cascade went all the way: the project row is gone (hard
		// delete) and the deletions row is gone (cron cleanup). Both
		// must hold for the next sweep tick to stop seeing this id.
		_, err = db.Query.FindProjectAnyById(h.Ctx, h.DB.RO(), proj.ID)
		require.True(t, db.IsNotFound(err), "expected project row removed, got err=%v", err)

		_, err = db.Query.FindDeletionByResource(h.Ctx, h.DB.RO(), db.FindDeletionByResourceParams{
			ResourceType: "project",
			ResourceID:   proj.ID,
		})
		require.True(t, db.IsNotFound(err), "expected deletions row removed, got err=%v", err)
	})

	t.Run("skips project whose delete_permanently_at is in the future", func(t *testing.T) {
		ws := h.Seed.CreateWorkspace(h.Ctx)
		proj := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
			ID:          uid.New(uid.ProjectPrefix),
			WorkspaceID: ws.ID,
			Name:        "future-project",
			Slug:        uid.New("slug"),
		})

		// One hour in the future relative to the test clock.
		futureT := fixed.Add(1 * time.Hour).UnixMilli()
		stampDueProject(t, h, ws.ID, proj.ID, futureT)

		_, err := callRunPermanentDelete(h, "future")
		require.NoError(t, err)

		// The deletions row must still be present at the original T.
		deletion, err := db.Query.FindDeletionByResource(h.Ctx, h.DB.RO(), db.FindDeletionByResourceParams{
			ResourceType: "project",
			ResourceID:   proj.ID,
		})
		require.NoError(t, err)
		require.Equal(t, futureT, deletion.DeletePermanentlyAt)
	})

	t.Run("skips live projects (deletion_id IS NULL)", func(t *testing.T) {
		ws := h.Seed.CreateWorkspace(h.Ctx)
		proj := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
			ID:          uid.New(uid.ProjectPrefix),
			WorkspaceID: ws.ID,
			Name:        "live-project",
			Slug:        uid.New("slug"),
		})

		_, err := callRunPermanentDelete(h, "live")
		require.NoError(t, err)

		// Row unchanged, still live (no deletion_id), still visible via
		// the normal find.
		row, err := db.Query.FindProjectById(h.Ctx, h.DB.RO(), proj.ID)
		require.NoError(t, err)
		require.False(t, row.DeletionID.Valid)
	})
}

// stampDueProject writes both halves of a soft delete: the deletions
// row and the project's deletion_id pointing at it. We bypass the
// MarkForDeletion VO chain because the cron sweep is the unit under
// test.
func stampDueProject(t *testing.T, h *harness.Harness, workspaceID, projectID string, atMs int64) {
	t.Helper()
	deletionID := string(uid.New(uid.DeletionPrefix))
	require.NoError(t, db.Query.InsertDeletion(h.Ctx, h.DB.RW(), db.InsertDeletionParams{
		ID:                  deletionID,
		WorkspaceID:         workspaceID,
		ResourceType:        "project",
		ResourceID:          projectID,
		DeletePermanentlyAt: atMs,
	}))
	_, err := db.Query.UpdateProjectDeletionId(h.Ctx, h.DB.RW(), db.UpdateProjectDeletionIdParams{
		ID:                 projectID,
		ExpectedDeletionID: sql.NullString{Valid: false},
		DeletionID:         sql.NullString{Valid: true, String: deletionID},
		UpdatedAt:          sql.NullInt64{Valid: true, Int64: atMs},
	})
	require.NoError(t, err)
}

func callRunPermanentDelete(h *harness.Harness, key string) (*hydrav1.RunPermanentDeleteResponse, error) {
	client := hydrav1.NewCronServiceIngressClient(h.Restate, key)
	return client.RunPermanentDelete().Request(h.Ctx, &hydrav1.RunPermanentDeleteRequest{})
}
