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

// FindVerifiedDomainClaimExcludingWorkspace is the only thing that makes
// contention span both domain tables, so it is pinned against real MySQL.
func TestFindVerifiedDomainClaimExcludingWorkspace(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	// Isolated so the schema is applied fresh, including portal_domains.
	mysqlCfg := containers.MySQLIsolated(t)
	database, err := New(mysqlCfg.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	insertCustom := func(workspaceID, domain string, status CustomDomainsVerificationStatus) string {
		id := uid.New(uid.DomainPrefix)
		require.NoError(t, database.InsertCustomDomain(ctx, InsertCustomDomainParams{
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
			CreatedAt:             time.Now().UnixMilli(),
		}))
		return id
	}
	insertPortal := func(workspaceID, domain string, status PortalDomainsVerificationStatus) string {
		id := uid.New(uid.PortalDomainPrefix)
		require.NoError(t, database.InsertPortalDomain(ctx, InsertPortalDomainParams{
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
			CreatedAt:             time.Now().UnixMilli(),
		}))
		return id
	}
	find := func(domain, workspaceID string) (FindVerifiedDomainClaimExcludingWorkspaceRow, error) {
		return database.FindVerifiedDomainClaimExcludingWorkspace(ctx, FindVerifiedDomainClaimExcludingWorkspaceParams{
			Domain:      domain,
			WorkspaceID: workspaceID,
		})
	}

	self := uid.New(uid.WorkspacePrefix)
	other := uid.New(uid.WorkspacePrefix)

	t.Run("a verified custom domain in another workspace", func(t *testing.T) {
		id := insertCustom(other, "custom.example.com", CustomDomainsVerificationStatusVerified)
		claim, err := find("custom.example.com", self)
		require.NoError(t, err)
		require.Equal(t, FindVerifiedDomainClaimExcludingWorkspaceRow{Source: "custom", ID: id, WorkspaceID: other}, claim)
	})

	t.Run("a verified portal domain in another workspace", func(t *testing.T) {
		id := insertPortal(other, "portal.example.com", PortalDomainsVerificationStatusVerified)
		claim, err := find("portal.example.com", self)
		require.NoError(t, err)
		require.Equal(t, FindVerifiedDomainClaimExcludingWorkspaceRow{Source: "portal", ID: id, WorkspaceID: other}, claim)
	})

	t.Run("the same workspace in either table is not a claim", func(t *testing.T) {
		insertCustom(self, "mine.example.com", CustomDomainsVerificationStatusVerified)
		insertPortal(self, "mine.example.com", PortalDomainsVerificationStatusVerified)
		_, err := find("mine.example.com", self)
		require.True(t, IsNotFound(err), "expected no claim, got %v", err)
	})

	t.Run("an unverified row is not a claim", func(t *testing.T) {
		insertCustom(other, "pending.example.com", CustomDomainsVerificationStatusPending)
		insertPortal(other, "pending.example.com", PortalDomainsVerificationStatusFailed)
		_, err := find("pending.example.com", self)
		require.True(t, IsNotFound(err), "expected no claim, got %v", err)
	})
}
