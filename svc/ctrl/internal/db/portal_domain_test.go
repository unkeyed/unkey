package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestPortalDomainUniqueness(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	// Isolated so the schema is applied fresh: the shared container skips
	// re-applying it once marked, and may predate portal_domains.
	mysqlCfg := containers.MySQLIsolated(t)
	database, err := New(mysqlCfg.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	insert := func(workspaceID, domain, targetCname string) error {
		return database.InsertPortalDomain(ctx, InsertPortalDomainParams{
			ID:                    uid.New(uid.PortalDomainPrefix),
			WorkspaceID:           workspaceID,
			PortalID:              uid.New(uid.PortalPrefix),
			Domain:                domain,
			VerificationStatus:    PortalDomainsVerificationStatusPending,
			VerificationToken:     uid.New(uid.TestPrefix),
			TargetCname:           targetCname,
			DomainConnectProvider: sql.NullString{},
			DomainConnectUrl:      sql.NullString{},
			InvocationID:          sql.NullString{},
			CreatedAt:             time.Now().UnixMilli(),
		})
	}

	workspaceA := uid.New(uid.WorkspacePrefix)
	workspaceB := uid.New(uid.WorkspacePrefix)
	domain := "portal.example.com"

	require.NoError(t, insert(workspaceA, domain, uid.New(uid.TestPrefix)))

	t.Run("same domain in the same workspace fails", func(t *testing.T) {
		err := insert(workspaceA, domain, uid.New(uid.TestPrefix))
		require.True(t, IsDuplicateKeyError(err), "expected duplicate key error, got %v", err)
	})

	t.Run("same domain in another workspace succeeds", func(t *testing.T) {
		require.NoError(t, insert(workspaceB, domain, uid.New(uid.TestPrefix)))
	})

	t.Run("duplicate target cname fails", func(t *testing.T) {
		targetCname := uid.New(uid.TestPrefix)
		require.NoError(t, insert(workspaceA, "a.example.com", targetCname))
		err := insert(workspaceB, "b.example.com", targetCname)
		require.True(t, IsDuplicateKeyError(err), "expected duplicate key error, got %v", err)
	})
}
