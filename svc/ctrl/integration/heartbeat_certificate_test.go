//go:build integration

package integration

import (
	"context"
	"database/sql"
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

// TestHeartbeatProvisionsCertificates verifies that the first Heartbeat for a
// brand-new cell auto-provisions its region and cell wildcard certificate
// records, and that a repeat Heartbeat is idempotent.
func TestHeartbeatProvisionsCertificates(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	const (
		bearer         = "test-bearer"
		regionalDomain = "unkey.cloud"
		platform       = "aws"
		regionName     = "ap-southeast-1"
		cellID         = "cell004"
	)
	// Matches the platform-ful format the frontline expects, e.g.
	// *.ap-southeast-1.aws.unkey.cloud.
	wildcardDomain := "*." + regionName + "." + platform + "." + regionalDomain
	cellWildcardDomain := "*." + cellID + "." + regionName + "." + platform + "." + regionalDomain

	heartbeat := func(svc *cluster.Service) {
		req := connect.NewRequest(&ctrlv1.HeartbeatRequest{
			Cluster: &ctrlv1.ClusterKey{CellId: cellID, Platform: platform, Region: regionName},
		})
		req.Header().Set("Authorization", "Bearer "+bearer)
		_, hbErr := svc.Heartbeat(ctx, req)
		require.NoError(t, hbErr)
	}

	svc := newHeartbeatService(t, h.DB, bearer, regionalDomain)

	// First heartbeat: registers the region and provisions the cert records.
	heartbeat(svc)

	domain, err := h.DB.FindCustomDomainByDomain(ctx, wildcardDomain)
	require.NoError(t, err)
	require.Equal(t, "unkey_internal", domain.WorkspaceID)
	require.Equal(t, db.CustomDomainsChallengeTypeDNS01, domain.ChallengeType)
	require.Equal(t, db.CustomDomainsVerificationStatusVerified, domain.VerificationStatus)

	require.Equal(t, 1, countCustomDomains(ctx, t, h.DB, wildcardDomain))
	require.Equal(t, 1, countAcmeChallenges(ctx, t, h.DB, domain.ID))
	cellDomain, err := h.DB.FindCustomDomainByDomain(ctx, cellWildcardDomain)
	require.NoError(t, err)
	require.Equal(t, 1, countCustomDomains(ctx, t, h.DB, cellWildcardDomain))
	require.Equal(t, 1, countAcmeChallenges(ctx, t, h.DB, cellDomain.ID))

	// Second heartbeat for the same region must not create duplicate records:
	// the existing challenge row makes EnsureInfraCertificate a no-op.
	heartbeat(svc)
	require.Equal(t, 1, countCustomDomains(ctx, t, h.DB, wildcardDomain))
	require.Equal(t, 1, countAcmeChallenges(ctx, t, h.DB, domain.ID))
	require.Equal(t, 1, countCustomDomains(ctx, t, h.DB, cellWildcardDomain))
	require.Equal(t, 1, countAcmeChallenges(ctx, t, h.DB, cellDomain.ID))

	// Recovery: if provisioning previously failed before writing the challenge
	// (region row exists, backstop missing), a later heartbeat must re-create it
	// rather than skip forever. Simulate the partial state by deleting the
	// challenge, then heartbeat from a fresh replica (cold cache) so the check
	// hits the DB rather than the first instance's cached "provisioned" entry.
	require.NoError(t, h.DB.DeleteAcmeChallengeByDomainID(ctx, domain.ID))
	require.Equal(t, 0, countAcmeChallenges(ctx, t, h.DB, domain.ID))

	heartbeat(newHeartbeatService(t, h.DB, bearer, regionalDomain))
	require.Equal(t, 1, countCustomDomains(ctx, t, h.DB, wildcardDomain))
	require.Equal(t, 1, countAcmeChallenges(ctx, t, h.DB, domain.ID))

	// Recovery from a terminal 'failed' challenge: once ProcessChallenge gives up
	// it marks the row 'failed', which the renewal cron skips (it matches only
	// 'waiting'/expiring-'verified') and nothing else resets. A heartbeat from a
	// fresh replica must flip it back to 'waiting' so issuance is retried rather
	// than stuck forever.
	require.NoError(t, h.DB.UpdateAcmeChallengeStatus(ctx, db.UpdateAcmeChallengeStatusParams{
		Status:    db.AcmeChallengesStatusFailed,
		UpdatedAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		DomainID:  domain.ID,
	}))

	heartbeat(newHeartbeatService(t, h.DB, bearer, regionalDomain))
	require.Equal(t, 1, countAcmeChallenges(ctx, t, h.DB, domain.ID))
	require.Equal(t, "waiting", challengeStatus(ctx, t, h.DB, domain.ID))
}

// newHeartbeatService builds an isolated ctrl service instance backed by the
// supplied database, modeling a separate replica or restarted process.
func newHeartbeatService(t *testing.T, database db.Database, bearer, regionalDomain string) *cluster.Service {
	t.Helper()

	topologyCache, err := cache.New(cache.Config[string, []db.FindDeploymentTopologyMinReplicasRow]{
		Fresh:    5 * time.Minute,
		Stale:    30 * time.Minute,
		MaxSize:  10,
		Resource: "test_topology",
		Clock:    clock.New(),
	})
	require.NoError(t, err)

	svc, err := cluster.New(cluster.Config{
		Database: database,
		// Points nowhere: triggering issuance is best-effort, so the failing
		// Send must not affect the DB records or the heartbeat response.
		Restate:        ingress.NewClient("http://127.0.0.1:1"),
		Bearer:         bearer,
		Clock:          clock.New(),
		TopologyCache:  topologyCache,
		InstanceEvents: batch.NewNoop[schema.InstanceEventV1](),
		RegionalDomain: regionalDomain,
	})
	require.NoError(t, err)
	return svc
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
