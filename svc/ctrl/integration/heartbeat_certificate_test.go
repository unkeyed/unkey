//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/services/cluster"

	"github.com/restatedev/sdk-go/ingress"
)

// TestHeartbeatProvisionsRegionCertificate verifies that the first Heartbeat for
// a brand-new region auto-provisions its wildcard certificate records (replacing
// the old manual bootstrap), and that a repeat Heartbeat is idempotent.
func TestHeartbeatProvisionsRegionCertificate(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	const (
		bearer         = "test-bearer"
		regionalDomain = "unkey.cloud"
		platform       = "aws"
		regionName     = "ap-southeast-1"
	)
	// Matches the platform-ful format the frontline expects, e.g.
	// *.ap-southeast-1.aws.unkey.cloud.
	wildcardDomain := "*." + regionName + "." + platform + "." + regionalDomain

	// newSvc builds a cluster service with a cold provisioned-cert cache. Each
	// instance models a separate ctrl replica (or a restarted process): a fresh
	// cache forces EnsureInfraCertificate to consult the DB, which is what proves
	// recovery is driven by durable state, not in-memory bookkeeping.
	newSvc := func() *cluster.Service {
		topologyCache, cacheErr := cache.New(cache.Config[string, []db.FindDeploymentTopologyMinReplicasRow]{
			Fresh:    5 * time.Minute,
			Stale:    30 * time.Minute,
			MaxSize:  10,
			Resource: "test_topology",
			Clock:    clock.New(),
		})
		require.NoError(t, cacheErr)

		svc, svcErr := cluster.New(cluster.Config{
			Database: h.DB,
			// Points nowhere: triggering issuance is best-effort, so the failing
			// Send must not affect the DB records or the heartbeat response.
			Restate:        ingress.NewClient("http://127.0.0.1:1"),
			Bearer:         bearer,
			Clock:          clock.New(),
			TopologyCache:  topologyCache,
			InstanceEvents: batch.NewNoop[schema.InstanceEventV1](),
			RegionalDomain: regionalDomain,
		})
		require.NoError(t, svcErr)
		return svc
	}

	heartbeat := func(svc *cluster.Service) {
		req := connect.NewRequest(&ctrlv1.HeartbeatRequest{
			Region: &ctrlv1.RegionKey{Platform: platform, Name: regionName},
		})
		req.Header().Set("Authorization", "Bearer "+bearer)
		_, hbErr := svc.Heartbeat(ctx, req)
		require.NoError(t, hbErr)
	}

	svc := newSvc()

	// First heartbeat: registers the region and provisions the cert records.
	heartbeat(svc)

	domain, err := h.DB.FindCustomDomainByDomain(ctx, wildcardDomain)
	require.NoError(t, err)
	require.Equal(t, "unkey_internal", domain.WorkspaceID)
	require.Equal(t, db.CustomDomainsChallengeTypeDNS01, domain.ChallengeType)
	require.Equal(t, db.CustomDomainsVerificationStatusVerified, domain.VerificationStatus)

	require.Equal(t, 1, countCustomDomains(ctx, t, h.DB, wildcardDomain))
	require.Equal(t, 1, countAcmeChallenges(ctx, t, h.DB, domain.ID))

	// Second heartbeat for the same region must not create duplicate records:
	// the existing challenge row makes EnsureInfraCertificate a no-op.
	heartbeat(svc)
	require.Equal(t, 1, countCustomDomains(ctx, t, h.DB, wildcardDomain))
	require.Equal(t, 1, countAcmeChallenges(ctx, t, h.DB, domain.ID))

	// Recovery: if provisioning previously failed before writing the challenge
	// (region row exists, backstop missing), a later heartbeat must re-create it
	// rather than skip forever. Simulate the partial state by deleting the
	// challenge, then heartbeat from a fresh replica (cold cache) so the check
	// hits the DB rather than the first instance's cached "provisioned" entry.
	_, err = h.DB.RW().ExecContext(ctx, "DELETE FROM acme_challenges WHERE domain_id = ?", domain.ID)
	require.NoError(t, err)
	require.Equal(t, 0, countAcmeChallenges(ctx, t, h.DB, domain.ID))

	heartbeat(newSvc())
	require.Equal(t, 1, countCustomDomains(ctx, t, h.DB, wildcardDomain))
	require.Equal(t, 1, countAcmeChallenges(ctx, t, h.DB, domain.ID))

	// Recovery from a terminal 'failed' challenge: once ProcessChallenge gives up
	// it marks the row 'failed', which the renewal cron skips (it matches only
	// 'waiting'/expiring-'verified') and nothing else resets. A heartbeat from a
	// fresh replica must flip it back to 'waiting' so issuance is retried rather
	// than stuck forever.
	_, err = h.DB.RW().ExecContext(ctx, "UPDATE acme_challenges SET status = 'failed' WHERE domain_id = ?", domain.ID)
	require.NoError(t, err)

	heartbeat(newSvc())
	require.Equal(t, 1, countAcmeChallenges(ctx, t, h.DB, domain.ID))
	require.Equal(t, "waiting", challengeStatus(ctx, t, h.DB, domain.ID))
}

func challengeStatus(ctx context.Context, t *testing.T, database db.Database, domainID string) string {
	t.Helper()
	var status string
	err := database.RO().QueryRowContext(ctx, "SELECT status FROM acme_challenges WHERE domain_id = ?", domainID).Scan(&status)
	require.NoError(t, err)
	return status
}

func countCustomDomains(ctx context.Context, t *testing.T, database db.Database, domain string) int {
	t.Helper()
	var n int
	err := database.RO().QueryRowContext(ctx, "SELECT COUNT(*) FROM custom_domains WHERE domain = ?", domain).Scan(&n)
	require.NoError(t, err)
	return n
}

func countAcmeChallenges(ctx context.Context, t *testing.T, database db.Database, domainID string) int {
	t.Helper()
	var n int
	err := database.RO().QueryRowContext(ctx, "SELECT COUNT(*) FROM acme_challenges WHERE domain_id = ?", domainID).Scan(&n)
	require.NoError(t, err)
	return n
}
