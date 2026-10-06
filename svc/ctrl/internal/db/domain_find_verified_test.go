package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

// Certificate issuance must act only on the verified claim of a hostname,
// whichever table holds it, so this is pinned against real MySQL.
func TestFindVerifiedDomainByHostname(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	database := certTestDB(t)
	now := time.Now().UnixMilli()
	find := func(domain string) (FindVerifiedDomainByHostnameRow, error) {
		return database.FindVerifiedDomainByHostname(ctx, FindVerifiedDomainByHostnameParams{Domain: domain})
	}

	t.Run("a verified deploy domain", func(t *testing.T) {
		ws := uid.New(uid.WorkspacePrefix)
		id := insertTestCustomDomain(t, database, ws, "deploy.example.com", CustomDomainsVerificationStatusVerified, now)
		row, err := find("deploy.example.com")
		require.NoError(t, err)
		require.Equal(t, FindVerifiedDomainByHostnameRow{ID: id, WorkspaceID: ws, Domain: "deploy.example.com"}, row)
	})

	t.Run("a verified portal domain", func(t *testing.T) {
		ws := uid.New(uid.WorkspacePrefix)
		id := insertTestPortalDomain(t, database, ws, "portal.example.com", PortalDomainsVerificationStatusVerified, now)
		row, err := find("portal.example.com")
		require.NoError(t, err)
		require.Equal(t, FindVerifiedDomainByHostnameRow{ID: id, WorkspaceID: ws, Domain: "portal.example.com"}, row)
	})

	t.Run("a pending claim beside a verified one is ignored", func(t *testing.T) {
		verifiedWS := uid.New(uid.WorkspacePrefix)
		// The pending row is newer, so only the status filter can exclude it.
		verified := insertTestPortalDomain(t, database, verifiedWS, "contested.example.com", PortalDomainsVerificationStatusVerified, now-1000)
		insertTestCustomDomain(t, database, uid.New(uid.WorkspacePrefix), "contested.example.com", CustomDomainsVerificationStatusPending, now)
		row, err := find("contested.example.com")
		require.NoError(t, err)
		require.Equal(t, verified, row.ID)
		require.Equal(t, verifiedWS, row.WorkspaceID)
	})

	t.Run("no verified row is not found", func(t *testing.T) {
		insertTestCustomDomain(t, database, uid.New(uid.WorkspacePrefix), "unverified.example.com", CustomDomainsVerificationStatusPending, now)
		insertTestPortalDomain(t, database, uid.New(uid.WorkspacePrefix), "unverified.example.com", PortalDomainsVerificationStatusFailed, now)
		_, err := find("unverified.example.com")
		require.True(t, IsNotFound(err), "expected not found, got %v", err)
	})

	t.Run("two verified rows resolve to the most recently updated", func(t *testing.T) {
		olderWS := uid.New(uid.WorkspacePrefix)
		newerWS := uid.New(uid.WorkspacePrefix)
		// The deploy row was created later but the portal row changed last.
		insertTestCustomDomain(t, database, olderWS, "twice.example.com", CustomDomainsVerificationStatusVerified, now-1000)
		newer := insertTestPortalDomain(t, database, newerWS, "twice.example.com", PortalDomainsVerificationStatusVerified, now-5000)
		require.NoError(t, database.UpdatePortalDomainVerificationStatus(ctx, UpdatePortalDomainVerificationStatusParams{
			ID:                 newer,
			VerificationStatus: PortalDomainsVerificationStatusVerified,
			UpdatedAt:          sql.NullInt64{Int64: now, Valid: true},
		}))
		row, err := find("twice.example.com")
		require.NoError(t, err)
		require.Equal(t, newer, row.ID)
		require.Equal(t, newerWS, row.WorkspaceID)
	})
}

// The ACME provider's lookup prefers an exact match in either table and falls
// back to a deploy wildcard, which is how infra wildcard certificates resolve.
func TestFindVerifiedDomainByHostnameOrWildcard(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	database := certTestDB(t)
	now := time.Now().UnixMilli()
	find := func(domain string) (FindVerifiedDomainByHostnameOrWildcardRow, error) {
		return database.FindVerifiedDomainByHostnameOrWildcard(ctx, FindVerifiedDomainByHostnameOrWildcardParams{
			Domain:         domain,
			WildcardDomain: "*." + domain,
		})
	}

	infraWS := uid.New(uid.WorkspacePrefix)
	wildcard := insertTestCustomDomain(t, database, infraWS, "*.region.example.com", CustomDomainsVerificationStatusVerified, now)

	t.Run("a wildcard resolves when no exact row exists", func(t *testing.T) {
		row, err := find("region.example.com")
		require.NoError(t, err)
		require.Equal(t, FindVerifiedDomainByHostnameOrWildcardRow{ID: wildcard, WorkspaceID: infraWS, Domain: "*.region.example.com"}, row)
	})

	t.Run("an exact portal row wins over the wildcard", func(t *testing.T) {
		ws := uid.New(uid.WorkspacePrefix)
		exact := insertTestPortalDomain(t, database, ws, "region.example.com", PortalDomainsVerificationStatusVerified, now-1000)
		row, err := find("region.example.com")
		require.NoError(t, err)
		require.Equal(t, exact, row.ID)
		require.Equal(t, ws, row.WorkspaceID)
	})

	t.Run("a pending exact claim beside a verified one is ignored", func(t *testing.T) {
		ws := uid.New(uid.WorkspacePrefix)
		verified := insertTestCustomDomain(t, database, ws, "app.example.com", CustomDomainsVerificationStatusVerified, now-1000)
		insertTestPortalDomain(t, database, uid.New(uid.WorkspacePrefix), "app.example.com", PortalDomainsVerificationStatusPending, now)
		row, err := find("app.example.com")
		require.NoError(t, err)
		require.Equal(t, verified, row.ID)
	})
}
