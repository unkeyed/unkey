package db

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
)

func TestCertificateHealthIncludesFailedAndPendingRenewals(t *testing.T) {
	config := containers.MySQLIsolated(t)
	database, err := New(config.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	ctx := t.Context()

	health, err := database.GetCertificateHealth(ctx)
	require.NoError(t, err)
	require.Equal(t, GetCertificateHealthRow{}, health)

	for _, row := range []struct {
		id     string
		status AcmeChallengesStatus
		expiry int64
		issued bool
	}{
		{id: "verified", status: AcmeChallengesStatusVerified, expiry: 1900000000000, issued: true},
		{id: "failed", status: AcmeChallengesStatusFailed, expiry: 1800000000000, issued: true},
		{id: "pending", status: AcmeChallengesStatusPending, expiry: 1700000000123, issued: true},
		{id: "unissued", status: AcmeChallengesStatusFailed, expiry: 1600000000000, issued: false},
		{id: "waiting", status: AcmeChallengesStatusWaiting, expiry: 0, issued: false},
	} {
		domain := row.id + ".example.com"
		require.NoError(t, database.InsertCustomDomain(ctx, InsertCustomDomainParams{
			ID: row.id, WorkspaceID: "ws_test", ProjectID: "prj_test", AppID: "app_test",
			EnvironmentID: "env_test", Domain: domain, ChallengeType: CustomDomainsChallengeTypeHTTP01,
			VerificationStatus: CustomDomainsVerificationStatusVerified, TargetCname: domain,
		}))
		require.NoError(t, database.InsertAcmeChallenge(ctx, InsertAcmeChallengeParams{
			DomainID: row.id, WorkspaceID: "ws_test", Status: row.status,
			ChallengeType: AcmeChallengesChallengeTypeHTTP01, ExpiresAt: row.expiry,
		}))
		if row.issued {
			require.NoError(t, database.InsertCertificate(ctx, InsertCertificateParams{
				ID: row.id, WorkspaceID: "ws_test", Hostname: domain,
				Certificate: "public-certificate", EncryptedPrivateKey: "encrypted-key",
			}))
		}
	}

	health, err = database.GetCertificateHealth(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(2), health.FailedChallenges)
	require.Equal(t, int64(1700000000123), health.EarliestExpiry)

	_, err = database.RW().ExecContext(ctx, "DELETE FROM custom_domains WHERE id = ?", "pending")
	require.NoError(t, err)
	health, err = database.GetCertificateHealth(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1800000000000), health.EarliestExpiry)

	_, err = database.RW().ExecContext(ctx, "DELETE FROM custom_domains WHERE id = ?", "failed")
	require.NoError(t, err)
	health, err = database.GetCertificateHealth(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), health.FailedChallenges)
	require.Equal(t, int64(1900000000000), health.EarliestExpiry)

	_, err = database.RW().ExecContext(ctx, "DELETE FROM custom_domains")
	require.NoError(t, err)
	health, err = database.GetCertificateHealth(ctx)
	require.NoError(t, err)
	require.Equal(t, GetCertificateHealthRow{}, health)
}
