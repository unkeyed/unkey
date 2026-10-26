package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestFindCustomDomainByDomainPrefersVerified pins the resolution rule for a
// hostname held by more than one workspace: the verified row wins, and among
// verified rows the one verified last wins.
func TestFindCustomDomainByDomainPrefersVerified(t *testing.T) {
	ctx := t.Context()
	mysqlCfg := containers.MySQLIsolated(t)
	database, err := New(mysqlCfg.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	insert := func(domain string, createdAt int64) string {
		t.Helper()
		id := uid.New(uid.DomainPrefix)
		require.NoError(t, database.InsertCustomDomain(ctx, InsertCustomDomainParams{
			ID:                    id,
			WorkspaceID:           uid.New(uid.WorkspacePrefix),
			ProjectID:             uid.New(uid.ProjectPrefix),
			AppID:                 uid.New(uid.AppPrefix),
			EnvironmentID:         uid.New(uid.EnvironmentPrefix),
			Domain:                domain,
			ChallengeType:         CustomDomainsChallengeTypeHTTP01,
			VerificationStatus:    CustomDomainsVerificationStatusPending,
			VerificationToken:     uid.New(uid.TestPrefix),
			TargetCname:           uid.New(uid.TestPrefix) + ".unkey-dns.com",
			DomainConnectProvider: sql.NullString{},
			DomainConnectUrl:      sql.NullString{},
			InvocationID:          sql.NullString{},
			CreatedAt:             createdAt,
		}))
		return id
	}

	setStatus := func(id string, status CustomDomainsVerificationStatus, at int64) {
		t.Helper()
		require.NoError(t, database.UpdateCustomDomainVerificationStatus(ctx, UpdateCustomDomainVerificationStatusParams{
			VerificationStatus: status,
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: at},
			ID:                 id,
		}))
	}

	now := time.Now().UnixMilli()

	t.Run("unverified squatter inserted first does not win", func(t *testing.T) {
		domain := uid.New(uid.TestPrefix) + ".example.com"
		insert(domain, now-1000)
		owner := insert(domain, now)
		setStatus(owner, CustomDomainsVerificationStatusVerified, now)

		got, err := database.FindCustomDomainByDomain(ctx, domain)
		require.NoError(t, err)
		require.Equal(t, owner, got.ID)
	})

	t.Run("unverified row touched after the owner does not win", func(t *testing.T) {
		domain := uid.New(uid.TestPrefix) + ".example.com"
		owner := insert(domain, now-2000)
		setStatus(owner, CustomDomainsVerificationStatusVerified, now-1000)
		squatter := insert(domain, now-500)
		setStatus(squatter, CustomDomainsVerificationStatusVerifying, now)

		got, err := database.FindCustomDomainByDomain(ctx, domain)
		require.NoError(t, err)
		require.Equal(t, owner, got.ID)
	})

	t.Run("row verified last wins during a takeover", func(t *testing.T) {
		domain := uid.New(uid.TestPrefix) + ".example.com"
		claimant := insert(domain, now-2000)
		previous := insert(domain, now-1000)
		setStatus(previous, CustomDomainsVerificationStatusVerified, now-500)
		setStatus(claimant, CustomDomainsVerificationStatusVerified, now)

		got, err := database.FindCustomDomainByDomain(ctx, domain)
		require.NoError(t, err)
		require.Equal(t, claimant, got.ID)

		wildcard, err := database.FindCustomDomainByDomainOrWildcard(ctx, FindCustomDomainByDomainOrWildcardParams{
			Domain:   domain,
			Domain_2: "*." + domain,
			Domain_3: domain,
		})
		require.NoError(t, err)
		require.Equal(t, claimant, wildcard.ID)
	})

	t.Run("single row still resolves", func(t *testing.T) {
		domain := uid.New(uid.TestPrefix) + ".example.com"
		only := insert(domain, now)

		got, err := database.FindCustomDomainByDomain(ctx, domain)
		require.NoError(t, err)
		require.Equal(t, only, got.ID)
	})
}
