package db_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// TestUpdateAcmeChallengeTryClaiming pins which rows a claim matches. Renewals
// arrive with the challenge already verified, so a claim that only matched
// waiting rows would report zero and abort issuance.
func TestUpdateAcmeChallengeTryClaiming(t *testing.T) {
	ctx := t.Context()

	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	seeder := seed.New(t, database, nil)
	seeder.Seed(ctx)
	workspaceID := seeder.Resources.UserWorkspace.ID

	project := seeder.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspaceID,
		Name:        "KEBAP",
		Slug:        "kebap-" + uid.New(uid.TestPrefix),
	})
	app := seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		Name:        "KEBAP",
		Slug:        "kebap-" + uid.New(uid.TestPrefix),
	})
	env := seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "production",
		Kind:        "production",
	})

	tests := []struct {
		name    string
		status  db.AcmeChallengesStatus
		claimed int64
	}{
		{name: "waiting is claimed", status: db.AcmeChallengesStatusWaiting, claimed: 1},
		{name: "verified renewal is claimed", status: db.AcmeChallengesStatusVerified, claimed: 1},
		{name: "pending is not reclaimed", status: db.AcmeChallengesStatusPending, claimed: 0},
		{name: "failed is not reclaimed", status: db.AcmeChallengesStatusFailed, claimed: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			domainID := uid.New(uid.DomainPrefix)
			domain := uid.DNS1035(12) + ".example.com"
			now := time.Now().UnixMilli()

			require.NoError(t, database.InsertCustomDomain(ctx, db.InsertCustomDomainParams{
				ID:                    domainID,
				WorkspaceID:           workspaceID,
				ProjectID:             project.ID,
				AppID:                 app.ID,
				EnvironmentID:         env.ID,
				Domain:                domain,
				ChallengeType:         db.CustomDomainsChallengeTypeHTTP01,
				VerificationStatus:    db.CustomDomainsVerificationStatusVerified,
				VerificationToken:     uid.Secure(24),
				TargetCname:           uid.DNS1035(16) + ".unkey-dns.com",
				DomainConnectProvider: sql.NullString{},
				DomainConnectUrl:      sql.NullString{},
				InvocationID:          sql.NullString{},
				CreatedAt:             now,
			}))
			require.NoError(t, database.InsertAcmeChallenge(ctx, db.InsertAcmeChallengeParams{
				WorkspaceID:   workspaceID,
				DomainID:      domainID,
				Token:         uid.Secure(24),
				Authorization: uid.Secure(24),
				Status:        test.status,
				ChallengeType: db.AcmeChallengesChallengeTypeHTTP01,
				CreatedAt:     now,
				UpdatedAt:     sql.NullInt64{Int64: now, Valid: true},
				ExpiresAt:     now,
			}))

			claimed, err := database.UpdateAcmeChallengeTryClaiming(ctx, db.UpdateAcmeChallengeTryClaimingParams{
				DomainID:  domainID,
				Status:    db.AcmeChallengesStatusPending,
				UpdatedAt: sql.NullInt64{Int64: now, Valid: true},
			})
			require.NoError(t, err)
			require.Equal(t, test.claimed, claimed)
		})
	}
}
