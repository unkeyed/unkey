package providers

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// The HTTP-01 provider records a challenge against the verified domain row a
// hostname resolves to, so these run against real MySQL to cover the query.
func TestPresentRecordsChallengeOnVerifiedDomain(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	database, err := db.New(containers.MySQLIsolated(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	now := time.Now().UnixMilli()
	insertCustom := func(workspaceID, domain string, status db.CustomDomainsVerificationStatus) string {
		id := uid.New(uid.DomainPrefix)
		require.NoError(t, database.InsertCustomDomain(ctx, db.InsertCustomDomainParams{
			ID:                    id,
			WorkspaceID:           workspaceID,
			ProjectID:             uid.New(uid.ProjectPrefix),
			AppID:                 uid.New(uid.AppPrefix),
			EnvironmentID:         uid.New(uid.EnvironmentPrefix),
			Domain:                domain,
			ChallengeType:         db.CustomDomainsChallengeTypeHTTP01,
			VerificationStatus:    status,
			VerificationToken:     uid.New(uid.TestPrefix),
			TargetCname:           uid.New(uid.TestPrefix),
			DomainConnectProvider: sql.NullString{},
			DomainConnectUrl:      sql.NullString{},
			InvocationID:          sql.NullString{},
			CreatedAt:             now,
		}))
		insertChallenge(t, database, workspaceID, id)
		return id
	}
	insertPortal := func(workspaceID, domain string, status db.PortalDomainsVerificationStatus) string {
		id := uid.New(uid.PortalDomainPrefix)
		require.NoError(t, database.InsertPortalDomain(ctx, db.InsertPortalDomainParams{
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
			CreatedAt:             now,
		}))
		insertChallenge(t, database, workspaceID, id)
		return id
	}

	domainCache, err := cache.New(cache.Config[string, db.FindVerifiedDomainByHostnameRow]{
		Fresh:    time.Minute,
		Stale:    time.Minute,
		MaxSize:  100,
		Resource: "test_domains",
		Clock:    clock.New(),
	})
	require.NoError(t, err)
	t.Cleanup(domainCache.Close)

	provider, err := NewHTTPProvider(HTTPConfig{DB: database, DomainCache: domainCache})
	require.NoError(t, err)

	t.Run("a pending claim beside a verified portal domain", func(t *testing.T) {
		pending := insertCustom(uid.New(uid.WorkspacePrefix), "contested.example.com", db.CustomDomainsVerificationStatusPending)
		verified := insertPortal(uid.New(uid.WorkspacePrefix), "contested.example.com", db.PortalDomainsVerificationStatusVerified)

		require.NoError(t, provider.Present("contested.example.com", "tok-contested", "auth-contested"))
		require.Equal(t, "auth-contested", challengeAuthorization(t, database, verified))
		require.Empty(t, challengeAuthorization(t, database, pending))
	})

	t.Run("an infra wildcard resolves through the wildcard arm", func(t *testing.T) {
		wildcard := insertCustom("unkey_internal", "*.region.example.com", db.CustomDomainsVerificationStatusVerified)

		require.NoError(t, provider.Present("region.example.com", "tok-wildcard", "auth-wildcard"))
		require.Equal(t, "auth-wildcard", challengeAuthorization(t, database, wildcard))
	})

	t.Run("no verified row is not found", func(t *testing.T) {
		insertPortal(uid.New(uid.WorkspacePrefix), "pending.example.com", db.PortalDomainsVerificationStatusPending)

		err := provider.Present("pending.example.com", "tok-pending", "auth-pending")
		require.True(t, db.IsNotFound(err), "expected not found, got %v", err)
	})

	t.Run("a miss before verification is not cached", func(t *testing.T) {
		id := insertPortal(uid.New(uid.WorkspacePrefix), "later.example.com", db.PortalDomainsVerificationStatusPending)
		require.Error(t, provider.Present("later.example.com", "tok-later", "auth-later"))

		require.NoError(t, database.UpdatePortalDomainVerificationStatus(ctx, db.UpdatePortalDomainVerificationStatusParams{
			ID:                 id,
			VerificationStatus: db.PortalDomainsVerificationStatusVerified,
			UpdatedAt:          sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		}))

		require.NoError(t, provider.Present("later.example.com", "tok-later", "auth-later"))
		require.Equal(t, "auth-later", challengeAuthorization(t, database, id))
	})
}

func insertChallenge(t *testing.T, database db.Database, workspaceID, domainID string) {
	t.Helper()
	now := time.Now().UnixMilli()
	require.NoError(t, database.InsertAcmeChallenge(context.Background(), db.InsertAcmeChallengeParams{
		WorkspaceID:   workspaceID,
		DomainID:      domainID,
		Token:         "",
		Authorization: "",
		Status:        db.AcmeChallengesStatusWaiting,
		ChallengeType: db.AcmeChallengesChallengeTypeHTTP01,
		CreatedAt:     now,
		UpdatedAt:     sql.NullInt64{Int64: now, Valid: true},
		ExpiresAt:     0,
	}))
}

func challengeAuthorization(t *testing.T, database db.Database, domainID string) string {
	t.Helper()
	var authorization string
	err := database.RO().QueryRowContext(context.Background(), "SELECT `authorization` FROM acme_challenges WHERE domain_id = ?", domainID).Scan(&authorization)
	require.NoError(t, err)
	return authorization
}
