package certificate_test

import (
	"context"
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
