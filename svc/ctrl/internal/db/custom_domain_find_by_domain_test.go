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

// TestFindCustomDomainByDomainPrefersVerified pins the resolution rule for a
// hostname held by more than one workspace.
//
// Uniqueness is (workspace_id, domain), so an unverified squatter and the
// verified owner both keep a row for the same name. Certificate issuance and
// ACME validation arrive with only a hostname, so the query itself has to pick
// the owner. Returning whichever row the optimizer reached first encrypted a
// certificate under a workspace that no longer owned the domain.
func TestFindCustomDomainByDomainPrefersVerified(t *testing.T) {
	ctx := context.Background()
	mysqlCfg := containers.MySQLIsolated(t)
	database, err := New(mysqlCfg.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	insert := func(workspaceID, domain, status string, createdAt int64) string {
		t.Helper()
		id := uid.New(uid.DomainPrefix)
		require.NoError(t, database.InsertCustomDomain(ctx, InsertCustomDomainParams{
			ID:                    id,
			WorkspaceID:           workspaceID,
			ProjectID:             uid.New(uid.ProjectPrefix),
			AppID:                 uid.New(uid.AppPrefix),
			EnvironmentID:         uid.New(uid.EnvironmentPrefix),
			Domain:                domain,
			ChallengeType:         CustomDomainsChallengeTypeHTTP01,
			VerificationStatus:    CustomDomainsVerificationStatus(status),
			VerificationToken:     uid.New(uid.TestPrefix),
			TargetCname:           uid.New(uid.TestPrefix) + ".unkey-dns.com",
			DomainConnectProvider: sql.NullString{},
			DomainConnectUrl:      sql.NullString{},
			InvocationID:          sql.NullString{},
			CreatedAt:             createdAt,
		}))
		return id
	}

	now := time.Now().UnixMilli()

	t.Run("unverified squatter inserted first does not win", func(t *testing.T) {
		domain := uid.New(uid.TestPrefix) + ".example.com"
		squatter := insert(uid.New(uid.WorkspacePrefix), domain, "pending", now-1000)
		owner := insert(uid.New(uid.WorkspacePrefix), domain, "verified", now)

		got, err := database.FindCustomDomainByDomain(ctx, domain)
		require.NoError(t, err)
		require.Equal(t, owner, got.ID, "issuance must resolve the verified owner, not the older squatter row")
		require.NotEqual(t, squatter, got.ID)
	})

	t.Run("newest verified row wins in the contention window", func(t *testing.T) {
		// Contention revoke marks the previous owner failed, but the window
		// before that leaves two verified rows. The newest is the claimant that
		// just proved control.
		domain := uid.New(uid.TestPrefix) + ".example.com"
		insert(uid.New(uid.WorkspacePrefix), domain, "verified", now-1000)
		newest := insert(uid.New(uid.WorkspacePrefix), domain, "verified", now)

		got, err := database.FindCustomDomainByDomain(ctx, domain)
		require.NoError(t, err)
		require.Equal(t, newest, got.ID)
	})

	t.Run("single row still resolves", func(t *testing.T) {
		domain := uid.New(uid.TestPrefix) + ".example.com"
		only := insert(uid.New(uid.WorkspacePrefix), domain, "pending", now)

		got, err := database.FindCustomDomainByDomain(ctx, domain)
		require.NoError(t, err)
		require.Equal(t, only, got.ID)
	})
}
