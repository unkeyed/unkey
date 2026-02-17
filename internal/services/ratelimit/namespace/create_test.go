package namespace

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func setupTest(t *testing.T) (Service, db.Database) {
	t.Helper()

	containers.StartAllServices(t)
	mysqlCfg := containers.MySQL(t)

	database, err := db.New(db.Config{
		PrimaryDSN:  mysqlCfg.FormatDSN(),
		ReadOnlyDSN: "",
	})
	require.NoError(t, err)

	c, err := caches.New(caches.Config{
		Broadcaster: nil,
		NodeID:      "",
		Clock:       clock.NewTestClock(),
	})
	require.NoError(t, err)

	audit, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)

	svc, err := New(Config{
		DB:        database,
		Cache:     c.RatelimitNamespace,
		Auditlogs: audit,
	})
	require.NoError(t, err)

	return svc, database
}

func createTestWorkspace(t *testing.T, database db.Database) string {
	t.Helper()
	wsID := uid.New("test_ws")
	err := db.Query.InsertWorkspace(context.Background(), database.RW(), db.InsertWorkspaceParams{
		ID:           wsID,
		OrgID:        uid.New("test_org"),
		Name:         uid.New("test"),
		Slug:         uid.New("slug"),
		CreatedAt:    time.Now().UnixMilli(),
		K8sNamespace: sql.NullString{Valid: true, String: uid.DNS1035()},
	})
	require.NoError(t, err)
	return wsID
}

func TestCreate(t *testing.T) {
	svc, database := setupTest(t)
	ctx := context.Background()
	wsID := createTestWorkspace(t, database)

	t.Run("creates namespace and writes audit log", func(t *testing.T) {
		name := uid.New("test_ns")
		audit := &AuditContext{
			ActorID:   uid.New("actor"),
			ActorName: "test-actor",
			ActorType: auditlog.RootKeyActor,
			RemoteIP:  "127.0.0.1",
			UserAgent: "test-agent",
		}

		ns, err := svc.Create(ctx, wsID, name, audit)
		require.NoError(t, err)
		require.NotEmpty(t, ns.ID)
		require.Equal(t, wsID, ns.WorkspaceID)
		require.Equal(t, name, ns.Name)

		// Verify the namespace exists in the DB
		row, err := db.Query.FindRatelimitNamespaceByName(ctx, database.RO(), db.FindRatelimitNamespaceByNameParams{
			WorkspaceID: wsID,
			Name:        name,
		})
		require.NoError(t, err)
		require.Equal(t, ns.ID, row.ID)
		require.Equal(t, name, row.Name)

		// Verify audit log was created
		auditTargets, err := db.Query.FindAuditLogTargetByID(ctx, database.RO(), ns.ID)
		require.NoError(t, err)
		require.Len(t, auditTargets, 1)
		require.Equal(t, string(auditlog.RatelimitNamespaceCreateEvent), auditTargets[0].AuditLog.Event)
		require.Equal(t, string(auditlog.RatelimitNamespaceResourceType), auditTargets[0].AuditLogTarget.Type)
	})

	t.Run("creates namespace without audit context", func(t *testing.T) {
		name := uid.New("test_ns")

		ns, err := svc.Create(ctx, wsID, name, nil)
		require.NoError(t, err)
		require.NotEmpty(t, ns.ID)
		require.Equal(t, name, ns.Name)

		// Verify namespace exists but no audit log
		row, err := db.Query.FindRatelimitNamespaceByName(ctx, database.RO(), db.FindRatelimitNamespaceByNameParams{
			WorkspaceID: wsID,
			Name:        name,
		})
		require.NoError(t, err)
		require.Equal(t, ns.ID, row.ID)

		auditTargets, err := db.Query.FindAuditLogTargetByID(ctx, database.RO(), ns.ID)
		require.NoError(t, err)
		require.Len(t, auditTargets, 0)
	})

	t.Run("duplicate name returns existing namespace", func(t *testing.T) {
		name := uid.New("test_ns")

		ns1, err := svc.Create(ctx, wsID, name, nil)
		require.NoError(t, err)

		// Creating same name again should succeed via re-fetch
		ns2, err := svc.Create(ctx, wsID, name, nil)
		require.NoError(t, err)
		require.Equal(t, ns1.ID, ns2.ID)
		require.Equal(t, ns1.Name, ns2.Name)
	})

	t.Run("can be fetched via Get after Create", func(t *testing.T) {
		name := uid.New("test_ns")

		created, err := svc.Create(ctx, wsID, name, nil)
		require.NoError(t, err)

		// Fetch by name
		fetched, found, err := svc.Get(ctx, wsID, name)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, created.ID, fetched.ID)

		// Fetch by ID
		fetched, found, err = svc.Get(ctx, wsID, created.ID)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, created.Name, fetched.Name)
	})
}

func TestCreateMany(t *testing.T) {
	svc, database := setupTest(t)
	ctx := context.Background()
	wsID := createTestWorkspace(t, database)

	t.Run("creates multiple namespaces", func(t *testing.T) {
		names := []string{
			uid.New("test_ns"),
			uid.New("test_ns"),
			uid.New("test_ns"),
		}

		audit := &AuditContext{
			ActorID:   uid.New("actor"),
			ActorName: "test-actor",
			ActorType: auditlog.RootKeyActor,
			RemoteIP:  "127.0.0.1",
			UserAgent: "test-agent",
		}

		created, err := svc.CreateMany(ctx, wsID, names, audit)
		require.NoError(t, err)
		require.Len(t, created, len(names))

		for _, name := range names {
			ns, ok := created[name]
			require.True(t, ok, "namespace %s should be in result", name)
			require.NotEmpty(t, ns.ID)
			require.Equal(t, name, ns.Name)
			require.Equal(t, wsID, ns.WorkspaceID)

			// Verify in DB
			row, err := db.Query.FindRatelimitNamespaceByName(ctx, database.RO(), db.FindRatelimitNamespaceByNameParams{
				WorkspaceID: wsID,
				Name:        name,
			})
			require.NoError(t, err)
			require.Equal(t, ns.ID, row.ID)
		}
	})

	t.Run("handles duplicate key race in batch", func(t *testing.T) {
		name := uid.New("test_ns")

		// Pre-create the namespace
		existing, err := svc.Create(ctx, wsID, name, nil)
		require.NoError(t, err)

		// CreateMany with the same name should succeed via re-fetch
		created, err := svc.CreateMany(ctx, wsID, []string{name}, nil)
		require.NoError(t, err)
		require.Len(t, created, 1)
		require.Equal(t, existing.ID, created[name].ID)
	})

	t.Run("all created namespaces fetchable via Get", func(t *testing.T) {
		names := []string{
			uid.New("test_ns"),
			uid.New("test_ns"),
		}

		created, err := svc.CreateMany(ctx, wsID, names, nil)
		require.NoError(t, err)

		for _, name := range names {
			ns, found, err := svc.Get(ctx, wsID, name)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, created[name].ID, ns.ID)
		}
	})
}
