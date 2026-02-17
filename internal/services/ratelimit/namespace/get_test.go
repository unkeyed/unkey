package namespace

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestGet(t *testing.T) {
	svc, database := setupTest(t)
	ctx := context.Background()
	wsID := createTestWorkspace(t, database)

	t.Run("returns not found for nonexistent namespace", func(t *testing.T) {
		_, found, err := svc.Get(ctx, wsID, "does_not_exist")
		require.NoError(t, err)
		require.False(t, found)
	})

	t.Run("finds namespace by name", func(t *testing.T) {
		name := uid.New("test_ns")
		id := uid.New(uid.RatelimitNamespacePrefix)

		err := db.Query.InsertRatelimitNamespace(ctx, database.RW(), db.InsertRatelimitNamespaceParams{
			ID:          id,
			WorkspaceID: wsID,
			Name:        name,
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		ns, found, err := svc.Get(ctx, wsID, name)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, id, ns.ID)
		require.Equal(t, name, ns.Name)
		require.Equal(t, wsID, ns.WorkspaceID)
	})

	t.Run("finds namespace by ID", func(t *testing.T) {
		name := uid.New("test_ns")
		id := uid.New(uid.RatelimitNamespacePrefix)

		err := db.Query.InsertRatelimitNamespace(ctx, database.RW(), db.InsertRatelimitNamespaceParams{
			ID:          id,
			WorkspaceID: wsID,
			Name:        name,
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		ns, found, err := svc.Get(ctx, wsID, id)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, id, ns.ID)
		require.Equal(t, name, ns.Name)
	})

	t.Run("does not find namespace from different workspace", func(t *testing.T) {
		otherWsID := createTestWorkspace(t, database)
		name := uid.New("test_ns")
		id := uid.New(uid.RatelimitNamespacePrefix)

		err := db.Query.InsertRatelimitNamespace(ctx, database.RW(), db.InsertRatelimitNamespaceParams{
			ID:          id,
			WorkspaceID: otherWsID,
			Name:        name,
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Should not find it in the original workspace
		_, found, err := svc.Get(ctx, wsID, name)
		require.NoError(t, err)
		require.False(t, found)
	})

	t.Run("returns overrides when present", func(t *testing.T) {
		name := uid.New("test_ns")
		nsID := uid.New(uid.RatelimitNamespacePrefix)

		err := db.Query.InsertRatelimitNamespace(ctx, database.RW(), db.InsertRatelimitNamespaceParams{
			ID:          nsID,
			WorkspaceID: wsID,
			Name:        name,
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Insert a direct override
		overrideID := uid.New(uid.RatelimitOverridePrefix)
		err = db.Query.InsertRatelimitOverride(ctx, database.RW(), db.InsertRatelimitOverrideParams{
			ID:          overrideID,
			WorkspaceID: wsID,
			NamespaceID: nsID,
			Identifier:  "user_123",
			Limit:       500,
			Duration:    60000,
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Invalidate cache to force DB reload with overrides
		//nolint:exhaustruct
		svc.Invalidate(ctx, wsID, db.FindRatelimitNamespace{
			ID:                nsID,
			WorkspaceID:       wsID,
			Name:              name,
			CreatedAtM:        0,
			UpdatedAtM:        sql.NullInt64{},
			DeletedAtM:        sql.NullInt64{},
			DirectOverrides:   nil,
			WildcardOverrides: nil,
		})

		ns, found, err := svc.Get(ctx, wsID, name)
		require.NoError(t, err)
		require.True(t, found)
		require.Contains(t, ns.DirectOverrides, "user_123")
		require.Equal(t, int64(500), ns.DirectOverrides["user_123"].Limit)
	})
}

func TestGetMany(t *testing.T) {
	svc, database := setupTest(t)
	ctx := context.Background()
	wsID := createTestWorkspace(t, database)

	t.Run("returns all found and missing names", func(t *testing.T) {
		name1 := uid.New("test_ns")
		name2 := uid.New("test_ns")
		missingName := uid.New("test_ns")

		for _, name := range []string{name1, name2} {
			err := db.Query.InsertRatelimitNamespace(ctx, database.RW(), db.InsertRatelimitNamespaceParams{
				ID:          uid.New(uid.RatelimitNamespacePrefix),
				WorkspaceID: wsID,
				Name:        name,
				CreatedAt:   time.Now().UnixMilli(),
			})
			require.NoError(t, err)
		}

		found, missing, err := svc.GetMany(ctx, wsID, []string{name1, name2, missingName})
		require.NoError(t, err)
		require.Len(t, found, 2)
		require.Len(t, missing, 1)
		require.Contains(t, found, name1)
		require.Contains(t, found, name2)
		require.Equal(t, missingName, missing[0])
	})

	t.Run("all missing returns empty found", func(t *testing.T) {
		found, missing, err := svc.GetMany(ctx, wsID, []string{
			uid.New("nope"),
			uid.New("nope"),
		})
		require.NoError(t, err)
		require.Len(t, found, 0)
		require.Len(t, missing, 2)
	})

	t.Run("all found returns empty missing", func(t *testing.T) {
		names := []string{uid.New("test_ns"), uid.New("test_ns")}
		for _, name := range names {
			err := db.Query.InsertRatelimitNamespace(ctx, database.RW(), db.InsertRatelimitNamespaceParams{
				ID:          uid.New(uid.RatelimitNamespacePrefix),
				WorkspaceID: wsID,
				Name:        name,
				CreatedAt:   time.Now().UnixMilli(),
			})
			require.NoError(t, err)
		}

		found, missing, err := svc.GetMany(ctx, wsID, names)
		require.NoError(t, err)
		require.Len(t, found, 2)
		require.Nil(t, missing)
	})
}
