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

// certTestDB opens an isolated database so the schema includes portal_domains.
func certTestDB(t *testing.T) Database {
	t.Helper()
	database, err := New(containers.MySQLIsolated(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	return database
}

func insertTestCustomDomain(t *testing.T, database Database, workspaceID, domain string, status CustomDomainsVerificationStatus, createdAt int64) string {
	t.Helper()
	id := uid.New(uid.DomainPrefix)
	require.NoError(t, database.InsertCustomDomain(context.Background(), InsertCustomDomainParams{
		ID:                    id,
		WorkspaceID:           workspaceID,
		ProjectID:             uid.New(uid.ProjectPrefix),
		AppID:                 uid.New(uid.AppPrefix),
		EnvironmentID:         uid.New(uid.EnvironmentPrefix),
		Domain:                domain,
		ChallengeType:         CustomDomainsChallengeTypeHTTP01,
		VerificationStatus:    status,
		VerificationToken:     uid.New(uid.TestPrefix),
		TargetCname:           uid.New(uid.TestPrefix),
		DomainConnectProvider: sql.NullString{},
		DomainConnectUrl:      sql.NullString{},
		InvocationID:          sql.NullString{},
		CreatedAt:             createdAt,
	}))
	return id
}

func insertTestPortalDomain(t *testing.T, database Database, workspaceID, domain string, status PortalDomainsVerificationStatus, createdAt int64) string {
	t.Helper()
	id := uid.New(uid.PortalDomainPrefix)
	require.NoError(t, database.InsertPortalDomain(context.Background(), InsertPortalDomainParams{
		ID:                    id,
		WorkspaceID:           workspaceID,
		PortalID:              uid.New(uid.PortalPrefix),
		Domain:                domain,
		VerificationStatus:    status,
		VerificationToken:     uid.New(uid.TestPrefix),
		TargetCname:           uid.New(uid.TestPrefix),
		DomainConnectProvider: sql.NullString{},
		DomainConnectUrl:      sql.NullString{},
		InvocationID:          sql.NullString{},
		CreatedAt:             createdAt,
	}))
	return id
}

func insertTestChallenge(t *testing.T, database Database, workspaceID, domainID string, status AcmeChallengesStatus, expiresAt int64) {
	t.Helper()
	now := time.Now().UnixMilli()
	require.NoError(t, database.InsertAcmeChallenge(context.Background(), InsertAcmeChallengeParams{
		WorkspaceID:   workspaceID,
		DomainID:      domainID,
		Token:         "",
		Authorization: "",
		Status:        status,
		ChallengeType: AcmeChallengesChallengeTypeHTTP01,
		CreatedAt:     now,
		UpdatedAt:     sql.NullInt64{Int64: now, Valid: true},
		ExpiresAt:     expiresAt,
	}))
}

var executableTypes = []AcmeChallengesChallengeType{AcmeChallengesChallengeTypeHTTP01, AcmeChallengesChallengeTypeDNS01}

// Characterization: deploy domains are listed when waiting or within 30 days of
// expiry, oldest domain first, and nothing else.
func TestListExecutableChallengesDeployDomains(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	database := certTestDB(t)
	ws := uid.New(uid.WorkspacePrefix)
	now := time.Now()

	waiting := insertTestCustomDomain(t, database, ws, "waiting.example.com", CustomDomainsVerificationStatusVerified, now.Add(-2*time.Hour).UnixMilli())
	insertTestChallenge(t, database, ws, waiting, AcmeChallengesStatusWaiting, 0)

	expiring := insertTestCustomDomain(t, database, ws, "expiring.example.com", CustomDomainsVerificationStatusVerified, now.Add(-time.Hour).UnixMilli())
	insertTestChallenge(t, database, ws, expiring, AcmeChallengesStatusVerified, now.Add(10*24*time.Hour).UnixMilli())

	fresh := insertTestCustomDomain(t, database, ws, "fresh.example.com", CustomDomainsVerificationStatusVerified, now.UnixMilli())
	insertTestChallenge(t, database, ws, fresh, AcmeChallengesStatusVerified, now.Add(60*24*time.Hour).UnixMilli())

	rows, err := database.ListExecutableChallenges(ctx, executableTypes)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, "waiting.example.com", rows[0].Domain)
	require.Equal(t, ws, rows[0].WorkspaceID)
	require.Equal(t, "expiring.example.com", rows[1].Domain)
}

// Renewal covers portal domains too, and only domains still verified: a
// challenge left behind by a failed domain row must not be issued.
func TestListExecutableChallengesAcrossDomainTables(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	database := certTestDB(t)
	ws := uid.New(uid.WorkspacePrefix)
	now := time.Now()

	deployWaiting := insertTestCustomDomain(t, database, ws, "deploy.example.com", CustomDomainsVerificationStatusVerified, now.Add(-4*time.Hour).UnixMilli())
	insertTestChallenge(t, database, ws, deployWaiting, AcmeChallengesStatusWaiting, 0)

	portalWaiting := insertTestPortalDomain(t, database, ws, "portal.example.com", PortalDomainsVerificationStatusVerified, now.Add(-3*time.Hour).UnixMilli())
	insertTestChallenge(t, database, ws, portalWaiting, AcmeChallengesStatusWaiting, 0)

	portalExpiring := insertTestPortalDomain(t, database, ws, "renew.portal.example.com", PortalDomainsVerificationStatusVerified, now.Add(-2*time.Hour).UnixMilli())
	insertTestChallenge(t, database, ws, portalExpiring, AcmeChallengesStatusVerified, now.Add(10*24*time.Hour).UnixMilli())

	portalFresh := insertTestPortalDomain(t, database, ws, "fresh.portal.example.com", PortalDomainsVerificationStatusVerified, now.Add(-time.Hour).UnixMilli())
	insertTestChallenge(t, database, ws, portalFresh, AcmeChallengesStatusVerified, now.Add(60*24*time.Hour).UnixMilli())

	deployFailed := insertTestCustomDomain(t, database, ws, "failed.example.com", CustomDomainsVerificationStatusFailed, now.UnixMilli())
	insertTestChallenge(t, database, ws, deployFailed, AcmeChallengesStatusWaiting, 0)

	portalFailed := insertTestPortalDomain(t, database, ws, "failed.portal.example.com", PortalDomainsVerificationStatusFailed, now.UnixMilli())
	insertTestChallenge(t, database, ws, portalFailed, AcmeChallengesStatusWaiting, 0)

	rows, err := database.ListExecutableChallenges(ctx, executableTypes)
	require.NoError(t, err)
	domains := make([]string, 0, len(rows))
	for _, row := range rows {
		require.Equal(t, ws, row.WorkspaceID)
		domains = append(domains, row.Domain)
	}
	require.Equal(t, []string{"deploy.example.com", "portal.example.com", "renew.portal.example.com"}, domains)
}
