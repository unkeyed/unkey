package certificate_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/certificate"
)

// TestProcessChallengeMissingDomainDoesNotRetry proves a challenge for a domain
// that no longer exists finishes. A domain deleted between the renewal sweep and
// this invocation would otherwise keep retrying its lookup, because restate.Run
// retries a non-terminal error.
func TestProcessChallengeMissingDomainDoesNotRetry(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)

	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	svc := certificate.New(certificate.Config{DB: database, Vault: nil})
	restateCfg := containers.Restate(t, hydrav1.NewCertificateServiceServer(svc))
	admin := restateadmin.New(restateadmin.Config{BaseURL: restateCfg.AdminURL, APIKey: ""})

	domain := uid.DNS1035(12) + ".example.com"
	sent, err := hydrav1.NewCertificateServiceIngressClient(restateCfg.IngressClient, domain).
		ProcessChallenge().
		Send(ctx, &hydrav1.ProcessChallengeRequest{
			WorkspaceId: uid.New(uid.WorkspacePrefix),
			Domain:      domain,
		})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		live, liveErr := admin.FindLiveInvocations(ctx, []string{sent.Id()})
		require.NoError(t, liveErr)
		return !live[sent.Id()]
	}, 20*time.Second, 250*time.Millisecond,
		"invocation %s is still retrying; a missing domain must be terminal", sent.Id())
}

func TestProcessChallengeRejectsWorkspaceThatLostTheDomain(t *testing.T) {
	ctx := t.Context()
	mysqlCfg := containers.MySQLIsolated(t)
	database, err := db.New(mysqlCfg.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	restateCfg := containers.Restate(t, hydrav1.NewCertificateServiceServer(certificate.New(certificate.Config{DB: database})))

	domain := uid.New(uid.TestPrefix) + ".example.com"
	now := time.Now().UnixMilli()

	insert := func(workspaceID string, status db.CustomDomainsVerificationStatus, at int64) string {
		t.Helper()
		id := uid.New(uid.DomainPrefix)
		require.NoError(t, database.InsertCustomDomain(ctx, db.InsertCustomDomainParams{
			ID:                    id,
			WorkspaceID:           workspaceID,
			ProjectID:             uid.New(uid.ProjectPrefix),
			AppID:                 uid.New(uid.AppPrefix),
			EnvironmentID:         uid.New(uid.EnvironmentPrefix),
			Domain:                domain,
			ChallengeType:         db.CustomDomainsChallengeTypeHTTP01,
			VerificationStatus:    db.CustomDomainsVerificationStatusPending,
			VerificationToken:     uid.New(uid.TestPrefix),
			TargetCname:           uid.New(uid.TestPrefix) + ".unkey-dns.com",
			DomainConnectProvider: sql.NullString{},
			DomainConnectUrl:      sql.NullString{},
			InvocationID:          sql.NullString{},
			CreatedAt:             at,
		}))
		require.NoError(t, database.UpdateCustomDomainVerificationStatus(ctx, db.UpdateCustomDomainVerificationStatusParams{
			VerificationStatus: status,
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: at},
			ID:                 id,
		}))
		return id
	}

	previousWorkspaceID := uid.New(uid.WorkspacePrefix)
	ownerWorkspaceID := uid.New(uid.WorkspacePrefix)
	insert(previousWorkspaceID, db.CustomDomainsVerificationStatusFailed, now-1000)
	ownerDomainID := insert(ownerWorkspaceID, db.CustomDomainsVerificationStatusVerified, now)
	require.NoError(t, database.InsertAcmeChallenge(ctx, db.InsertAcmeChallengeParams{
		WorkspaceID:   ownerWorkspaceID,
		DomainID:      ownerDomainID,
		Token:         "",
		Authorization: "",
		Status:        db.AcmeChallengesStatusWaiting,
		ChallengeType: db.AcmeChallengesChallengeTypeHTTP01,
		CreatedAt:     now,
		UpdatedAt:     sql.NullInt64{Valid: true, Int64: now},
		ExpiresAt:     0,
	}))

	requestCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_, err = hydrav1.NewCertificateServiceIngressClient(restateCfg.IngressClient, domain).
		ProcessChallenge().
		Request(requestCtx, &hydrav1.ProcessChallengeRequest{WorkspaceId: previousWorkspaceID, Domain: domain})
	require.ErrorContains(t, err, "does not own domain")

	challenge, err := database.FindAcmeChallengeByDomain(ctx, domain)
	require.NoError(t, err)
	require.Equal(t, ownerDomainID, challenge.DomainID)
	require.Equal(t, db.AcmeChallengesStatusWaiting, challenge.Status, "the owner's challenge must not be claimed")
}
