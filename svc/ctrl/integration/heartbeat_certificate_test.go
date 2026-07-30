//go:build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/uid"
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

	heartbeat := func(svc *cluster.Service) {
		req := connect.NewRequest(&ctrlv1.HeartbeatRequest{
			Cluster: &ctrlv1.ClusterKey{CellId: "cell004", Platform: platform, Region: regionName},
		})
		req.Header().Set("Authorization", "Bearer "+bearer)
		_, hbErr := svc.Heartbeat(ctx, req)
		require.NoError(t, hbErr)
	}

	svc := newHeartbeatService(t, h.DB, bearer, regionalDomain)

	// First heartbeat: registers the region and provisions the cert records.
	heartbeat(svc)

	registeredCluster, err := h.DB.FindCluster(ctx, db.FindClusterParams{
		CellID:   sql.NullString{String: "cell004", Valid: true},
		Platform: platform,
		Region:   regionName,
	})
	require.NoError(t, err)
	require.Equal(t, sql.NullString{String: "cell004", Valid: true}, registeredCluster.Cluster.CellID)

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

	heartbeat(newHeartbeatService(t, h.DB, bearer, regionalDomain))
	require.Equal(t, 1, countCustomDomains(ctx, t, h.DB, wildcardDomain))
	require.Equal(t, 1, countAcmeChallenges(ctx, t, h.DB, domain.ID))

	// Recovery from a terminal 'failed' challenge: once ProcessChallenge gives up
	// it marks the row 'failed', which the renewal cron skips (it matches only
	// 'waiting'/expiring-'verified') and nothing else resets. A heartbeat from a
	// fresh replica must flip it back to 'waiting' so issuance is retried rather
	// than stuck forever.
	_, err = h.DB.RW().ExecContext(ctx, "UPDATE acme_challenges SET status = 'failed' WHERE domain_id = ?", domain.ID)
	require.NoError(t, err)

	heartbeat(newHeartbeatService(t, h.DB, bearer, regionalDomain))
	require.Equal(t, 1, countAcmeChallenges(ctx, t, h.DB, domain.ID))
	require.Equal(t, "waiting", challengeStatus(ctx, t, h.DB, domain.ID))
}

// TestHeartbeatPreservesClusterIdentityAcrossServiceInstances guarantees that a
// legacy row can claim one globally unique cell ID which is then immutable.
func TestHeartbeatPreservesClusterIdentityAcrossServiceInstances(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	const (
		bearer   = "test-bearer"
		platform = "aws"
	)
	regionName := uid.New("test_region")
	oldKey := &ctrlv1.ClusterKey{CellId: uid.New("cell"), Platform: platform, Region: regionName}
	newKey := &ctrlv1.ClusterKey{CellId: uid.New("cell"), Platform: platform, Region: regionName}
	oldNotFound := fmt.Sprintf("cluster %s/%s/%s not found", oldKey.GetCellId(), platform, regionName)
	newNotFound := fmt.Sprintf("cluster %s/%s/%s not found", newKey.GetCellId(), platform, regionName)

	heartbeat := func(svc *cluster.Service, key *ctrlv1.ClusterKey) {
		req := connect.NewRequest(&ctrlv1.HeartbeatRequest{Cluster: key})
		req.Header().Set("Authorization", "Bearer "+bearer)
		_, err := svc.Heartbeat(ctx, req)
		require.NoError(t, err)
	}
	getDesiredState := func(svc *cluster.Service, key *ctrlv1.ClusterKey) error {
		req := connect.NewRequest(&ctrlv1.GetDesiredDeploymentStateRequest{
			Cluster:      key,
			DeploymentId: "missing",
		})
		req.Header().Set("Authorization", "Bearer "+bearer)
		_, err := svc.GetDesiredDeploymentState(ctx, req)
		return err
	}

	createLegacyCluster(t, ctx, h.DB, platform, regionName)

	firstReplica := newHeartbeatService(t, h.DB, bearer, "")
	heartbeat(firstReplica, oldKey)
	// Resolve once before replacement so this test catches process-local caches.
	warmErr := getDesiredState(firstReplica, oldKey)
	require.Error(t, warmErr)
	require.NotContains(t, warmErr.Error(), oldNotFound)

	secondReplica := newHeartbeatService(t, h.DB, bearer, "")
	replacementReq := connect.NewRequest(&ctrlv1.HeartbeatRequest{Cluster: newKey})
	replacementReq.Header().Set("Authorization", "Bearer "+bearer)
	_, replacementErr := secondReplica.Heartbeat(ctx, replacementReq)
	require.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(replacementErr))

	oldIdentityErr := getDesiredState(firstReplica, oldKey)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(oldIdentityErr))
	require.NotContains(t, oldIdentityErr.Error(), oldNotFound)

	newIdentityErr := getDesiredState(firstReplica, newKey)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(newIdentityErr))
	require.ErrorContains(t, newIdentityErr, newNotFound)

	duplicateCellKey := &ctrlv1.ClusterKey{
		CellId:   oldKey.GetCellId(),
		Platform: platform,
		Region:   uid.New("test_region"),
	}
	duplicateCellReq := connect.NewRequest(&ctrlv1.HeartbeatRequest{Cluster: duplicateCellKey})
	duplicateCellReq.Header().Set("Authorization", "Bearer "+bearer)
	_, duplicateCellErr := secondReplica.Heartbeat(ctx, duplicateCellReq)
	require.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(duplicateCellErr))
}

// TestHeartbeatSerializesConcurrentCellClaims guarantees that exactly one cell
// can claim a legacy cluster when different ctrl replicas heartbeat concurrently.
func TestHeartbeatSerializesConcurrentCellClaims(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	const (
		bearer   = "test-bearer"
		platform = "aws"
	)
	regionName := uid.New("test_region")
	createLegacyCluster(t, ctx, h.DB, platform, regionName)

	keys := []*ctrlv1.ClusterKey{
		{CellId: uid.New("cell"), Platform: platform, Region: regionName},
		{CellId: uid.New("cell"), Platform: platform, Region: regionName},
	}
	services := []*cluster.Service{
		newHeartbeatService(t, h.DB, bearer, ""),
		newHeartbeatService(t, h.DB, bearer, ""),
	}
	type result struct {
		key *ctrlv1.ClusterKey
		err error
	}

	start := make(chan struct{})
	results := make(chan result, len(keys))
	var wg sync.WaitGroup
	for i := range keys {
		wg.Add(1)
		go func(key *ctrlv1.ClusterKey, svc *cluster.Service) {
			defer wg.Done()
			<-start
			req := connect.NewRequest(&ctrlv1.HeartbeatRequest{Cluster: key})
			req.Header().Set("Authorization", "Bearer "+bearer)
			_, err := svc.Heartbeat(ctx, req)
			results <- result{key: key, err: err}
		}(keys[i], services[i])
	}
	close(start)
	wg.Wait()
	close(results)

	var winner *ctrlv1.ClusterKey
	conflicts := 0
	for heartbeatResult := range results {
		if heartbeatResult.err == nil {
			require.Nil(t, winner, "more than one cell claimed the cluster")
			winner = heartbeatResult.key
			continue
		}
		require.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(heartbeatResult.err))
		conflicts++
	}
	require.NotNil(t, winner)
	require.Equal(t, 1, conflicts)

	registered, err := h.DB.FindCluster(ctx, db.FindClusterParams{
		CellID:   sql.NullString{String: winner.GetCellId(), Valid: true},
		Platform: winner.GetPlatform(),
		Region:   winner.GetRegion(),
	})
	require.NoError(t, err)
	require.Equal(t, sql.NullString{String: winner.GetCellId(), Valid: true}, registered.Cluster.CellID)
}

func createLegacyCluster(t *testing.T, ctx context.Context, database db.Database, platform, region string) {
	t.Helper()

	require.NoError(t, database.UpsertRegion(ctx, db.UpsertRegionParams{
		ID:       uid.New(uid.RegionPrefix),
		Name:     region,
		Platform: platform,
	}))
	legacyRegion, err := database.FindRegionByPlatformAndName(ctx, db.FindRegionByPlatformAndNameParams{
		Platform: platform,
		Name:     region,
	})
	require.NoError(t, err)
	require.NoError(t, database.UpsertCluster(ctx, db.UpsertClusterParams{
		ID:              uid.New(uid.ClusterPrefix),
		CellID:          sql.NullString{},
		RegionID:        legacyRegion.ID,
		LastHeartbeatAt: uint64(time.Now().UnixMilli()),
	}))
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
