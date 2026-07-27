package projects_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/projects"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

// TestEnsureDefaultProject guarantees ownership writers reuse an exact default,
// create a protected default when absent, and converge after concurrent creation.
func TestEnsureDefaultProject(t *testing.T) {
	t.Run("returns existing exact default", func(t *testing.T) {
		h := testutil.NewHarness(t)
		ctx := context.Background()
		workspaceID := h.Resources().UserWorkspace.ID
		existingID, err := db.Query.FindDefaultProjectByWorkspaceID(ctx, h.DB.RW(), workspaceID)
		require.NoError(t, err)

		projectID, err := db.TxWithResultRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (string, error) {
			return projects.EnsureDefaultProject(ctx, tx, workspaceID)
		})
		require.NoError(t, err)
		require.Equal(t, existingID, projectID)
	})

	t.Run("creates protected default when absent", func(t *testing.T) {
		h := testutil.NewHarness(t)
		ctx := context.Background()
		workspaceID := h.Resources().UserWorkspace.ID
		deleteDefaultProject(t, ctx, h.DB, workspaceID)

		projectID, err := db.TxWithResultRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (string, error) {
			return projects.EnsureDefaultProject(ctx, tx, workspaceID)
		})
		require.NoError(t, err)

		project, err := db.Query.FindProjectById(ctx, h.DB.RO(), projectID)
		require.NoError(t, err)
		require.Equal(t, workspaceID, project.WorkspaceID)
		require.Equal(t, "Default", project.Name)
		require.Equal(t, "default", project.Slug)
		require.Equal(t, sql.NullBool{Bool: true, Valid: true}, project.DeleteProtection)
	})

	t.Run("observes concurrently created default", func(t *testing.T) {
		h := testutil.NewHarness(t)
		ctx := context.Background()
		workspaceID := h.Resources().UserWorkspace.ID
		deleteDefaultProject(t, ctx, h.DB, workspaceID)

		initialReadResult := make(chan error, 1)
		concurrentCreateDone := make(chan struct{})
		projectIDResult := make(chan string, 1)
		errResult := make(chan error, 1)

		go func() {
			projectID, err := db.TxWithResultRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (string, error) {
				_, findErr := db.Query.FindDefaultProjectByWorkspaceID(ctx, tx, workspaceID)
				initialReadResult <- findErr
				if !db.IsNotFound(findErr) {
					return "", findErr
				}
				<-concurrentCreateDone
				return projects.EnsureDefaultProject(ctx, tx, workspaceID)
			})
			projectIDResult <- projectID
			errResult <- err
		}()

		require.True(t, db.IsNotFound(<-initialReadResult))
		concurrentID, err := db.TxWithResultRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (string, error) {
			return projects.EnsureDefaultProject(ctx, tx, workspaceID)
		})
		require.NoError(t, err)
		close(concurrentCreateDone)

		require.NoError(t, <-errResult)
		require.Equal(t, concurrentID, <-projectIDResult)
	})

	t.Run("rejects differently cased default", func(t *testing.T) {
		h := testutil.NewHarness(t)
		ctx := context.Background()
		workspaceID := h.Resources().UserWorkspace.ID
		deleteDefaultProject(t, ctx, h.DB, workspaceID)

		err := db.Query.InsertProject(ctx, h.DB.RW(), db.InsertProjectParams{
			ID:               uid.New(uid.ProjectPrefix),
			WorkspaceID:      workspaceID,
			Name:             "Default",
			Slug:             "Default",
			DeleteProtection: sql.NullBool{Bool: true, Valid: true},
			CreatedAt:        time.Now().UnixMilli(),
			UpdatedAt:        sql.NullInt64{},
		})
		require.NoError(t, err)

		_, err = db.TxWithResultRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (string, error) {
			return projects.EnsureDefaultProject(ctx, tx, workspaceID)
		})
		require.Error(t, err)
	})
}

func deleteDefaultProject(t *testing.T, ctx context.Context, database db.Database, workspaceID string) {
	t.Helper()

	projectID, err := db.Query.FindDefaultProjectByWorkspaceID(ctx, database.RW(), workspaceID)
	require.NoError(t, err)
	require.NoError(t, db.Query.DeleteProjectById(ctx, database.RW(), projectID))
}
