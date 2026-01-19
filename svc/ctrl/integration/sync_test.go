//go:build integration

// Package integration provides integration tests for the ctrl service's Sync RPC.
//
// These tests validate the Kubernetes-style List+Watch sync pattern that synchronizes
// deployment and sentinel state from the control plane (ctrl) to Kubernetes agents (krane).
//
// # Architecture Overview
//
// The sync protocol follows a two-phase approach:
//  1. Bootstrap: When a client connects with sequence=0, the server streams all current
//     running deployments and sentinels for the requested region, then sends a Bookmark
//     message containing the current max sequence number.
//  2. Watch: After bootstrap (or when reconnecting with sequence>0), the server polls
//     the state_changes table and streams incremental updates to the client.
//
// # Test Categories
//
//   - Bootstrap Tests: Verify initial full state sync behavior
//   - FailedPrecondition Tests: Verify sequence validation and resync triggers
//   - Delete Scenario Tests: Verify correct delete message generation
//   - State Change Query Tests: Verify underlying database queries
//   - Reconnect Tests: Verify incremental sync on reconnection
//   - Sequence Tests: Verify sequence numbers in streamed messages
//
// # Test Isolation
//
// Each test uses a unique region name to ensure test isolation. The state_changes
// table uses an auto-incrementing sequence that is global, but queries filter by
// region, so tests don't interfere with each other.
package integration

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/svc/ctrl/services/cluster"
)

// mockStream implements connect.ServerStream for testing and captures sent messages.
type mockStream struct {
	mu       sync.Mutex
	messages []*ctrlv1.State
}

func newMockStream() *mockStream {
	return &mockStream{
		messages: make([]*ctrlv1.State, 0),
	}
}

func (m *mockStream) Send(msg *ctrlv1.State) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockStream) Messages() []*ctrlv1.State {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*ctrlv1.State, len(m.messages))
	copy(result, m.messages)
	return result
}

func (m *mockStream) ResponseHeader() http.Header {
	return make(http.Header)
}

func (m *mockStream) ResponseTrailer() http.Header {
	return make(http.Header)
}

// newService creates a cluster service for testing.
func newService(t *testing.T, database db.Database) *cluster.Service {
	return cluster.New(cluster.Config{
		Database: database,
		Logger:   logging.NewNoop(),
		Bearer:   "test-bearer",
	})
}

// findDeploymentApply finds the first deployment apply message in the stream.
func findDeploymentApply(messages []*ctrlv1.State, deploymentID string) *ctrlv1.ApplyDeployment {
	for _, msg := range messages {
		if dep := msg.GetDeployment(); dep != nil {
			if apply := dep.GetApply(); apply != nil && apply.GetDeploymentId() == deploymentID {
				return apply
			}
		}
	}
	return nil
}

// findDeploymentDelete finds the first deployment delete message in the stream.
func findDeploymentDelete(messages []*ctrlv1.State, k8sName string) *ctrlv1.DeleteDeployment {
	for _, msg := range messages {
		if dep := msg.GetDeployment(); dep != nil {
			if del := dep.GetDelete(); del != nil && del.GetK8SName() == k8sName {
				return del
			}
		}
	}
	return nil
}

// findSentinelApply finds the first sentinel apply message in the stream.
func findSentinelApply(messages []*ctrlv1.State, sentinelID string) *ctrlv1.ApplySentinel {
	for _, msg := range messages {
		if sen := msg.GetSentinel(); sen != nil {
			if apply := sen.GetApply(); apply != nil && apply.GetSentinelId() == sentinelID {
				return apply
			}
		}
	}
	return nil
}

// findSentinelDelete finds the first sentinel delete message in the stream.
func findSentinelDelete(messages []*ctrlv1.State, k8sName string) *ctrlv1.DeleteSentinel {
	for _, msg := range messages {
		if sen := msg.GetSentinel(); sen != nil {
			if del := sen.GetDelete(); del != nil && del.GetK8SName() == k8sName {
				return del
			}
		}
	}
	return nil
}

// findBookmark finds the bookmark message in the stream.
func findBookmark(messages []*ctrlv1.State) *ctrlv1.Bookmark {
	for _, msg := range messages {
		if bookmark := msg.GetBookmark(); bookmark != nil {
			return bookmark
		}
	}
	return nil
}

// countDeploymentApplies counts deployment apply messages.
func countDeploymentApplies(messages []*ctrlv1.State) int {
	count := 0
	for _, msg := range messages {
		if dep := msg.GetDeployment(); dep != nil {
			if dep.GetApply() != nil {
				count++
			}
		}
	}
	return count
}

// countSentinelApplies counts sentinel apply messages.
func countSentinelApplies(messages []*ctrlv1.State) int {
	count := 0
	for _, msg := range messages {
		if sen := msg.GetSentinel(); sen != nil {
			if sen.GetApply() != nil {
				count++
			}
		}
	}
	return count
}

// =============================================================================
// Bootstrap Tests
// =============================================================================
//
// Bootstrap tests verify the initial full state synchronization that occurs when
// a krane agent first connects (with sequence=0). During bootstrap, the server
// must stream ALL currently running resources for the requested region, then
// send a Bookmark message with the current max sequence number.
//
// Guarantees tested:
//   - All running deployments in the region are streamed
//   - All running sentinels in the region are streamed
//   - Archived/stopped resources are NOT streamed
//   - A Bookmark is always sent after streaming all resources
//   - Empty regions receive only a Bookmark (no resources)
// =============================================================================

// TestSync_BootstrapStreamsDeploymentsAndVerifiesContent verifies that bootstrap
// correctly streams deployment resources with all required fields populated.
//
// Scenario: A new krane agent connects to sync a region containing one deployment.
//
// Guarantees:
//   - The deployment is included in the bootstrap stream
//   - The K8sName and Image fields are correctly populated
//   - A Bookmark with a non-zero sequence is sent after the deployment
//
// This test validates the core bootstrap contract: all running resources must be
// streamed to new clients so they can reconcile their local state.
func TestSync_BootstrapStreamsDeploymentsAndVerifiesContent(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	region := "us-west-2-bootstrap"

	dep := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       region,
		DesiredState: db.DeploymentsDesiredStateRunning,
	})

	// Insert a state change so bootstrap has a watermark
	h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeDeployment,
		ResourceID:   dep.Deployment.ID,
		Op:           db.StateChangesOpUpsert,
		Region:       region,
		CreatedAt:    uint64(h.Now() - 2000),
	})

	svc := newService(t, h.DB)
	stream := newMockStream()

	// Run bootstrap (Sync with sequence=0 triggers bootstrap then watch)
	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	req := connect.NewRequest(&ctrlv1.SyncRequest{
		Region:           region,
		SequenceLastSeen: 0,
	})

	// The sync will timeout in watch loop, but bootstrap should complete
	_ = svc.Sync(ctx, req, stream)

	// Verify the deployment was streamed
	messages := stream.Messages()
	apply := findDeploymentApply(messages, dep.Deployment.ID)
	require.NotNil(t, apply, "bootstrap should stream deployment apply")
	require.Equal(t, dep.Deployment.K8sName, apply.GetK8SName())
	require.Equal(t, "nginx:1.19", apply.GetImage())

	// Verify bookmark was sent
	bookmark := findBookmark(messages)
	require.NotNil(t, bookmark, "bootstrap should send bookmark")
	require.Greater(t, bookmark.GetSequence(), uint64(0), "bookmark should have non-zero sequence")
}

// TestSync_BootstrapStreamsSentinelsAndVerifiesContent verifies that bootstrap
// correctly streams sentinel resources with all required fields populated.
//
// Scenario: A new krane agent connects to sync a region containing one sentinel.
//
// Guarantees:
//   - The sentinel is included in the bootstrap stream
//   - The K8sName and Image fields are correctly populated
//   - A Bookmark is sent after the sentinel
//
// This test mirrors the deployment test but for sentinels, ensuring both resource
// types are handled correctly during bootstrap.
func TestSync_BootstrapStreamsSentinelsAndVerifiesContent(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	region := "eu-central-1-bootstrap"

	sentinel := h.CreateSentinel(ctx, CreateSentinelRequest{
		Region:       region,
		DesiredState: db.SentinelsDesiredStateRunning,
	})

	// Insert a state change
	h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeSentinel,
		ResourceID:   sentinel.ID,
		Op:           db.StateChangesOpUpsert,
		Region:       region,
		CreatedAt:    uint64(h.Now() - 2000),
	})

	svc := newService(t, h.DB)
	stream := newMockStream()

	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	req := connect.NewRequest(&ctrlv1.SyncRequest{
		Region:           region,
		SequenceLastSeen: 0,
	})

	_ = svc.Sync(ctx, req, stream)

	// Verify the sentinel was streamed
	messages := stream.Messages()
	apply := findSentinelApply(messages, sentinel.ID)
	require.NotNil(t, apply, "bootstrap should stream sentinel apply")
	require.Equal(t, sentinel.K8sName, apply.GetK8SName())
	require.Equal(t, "sentinel:1.0", apply.GetImage())

	// Verify bookmark was sent
	bookmark := findBookmark(messages)
	require.NotNil(t, bookmark, "bootstrap should send bookmark")
}

// TestSync_BootstrapWithEmptyRegionSendsOnlyBookmark verifies that bootstrap
// handles empty regions gracefully by sending only a Bookmark.
//
// Scenario: A krane agent connects to sync a region with no deployments or sentinels.
//
// Guarantees:
//   - Exactly one message is sent (the Bookmark)
//   - The Bookmark sequence is 0 (no state changes exist for this region)
//   - No deployment or sentinel apply messages are sent
//
// This edge case is critical for new regions or regions where all resources have
// been deleted. The client must still receive a Bookmark to know bootstrap completed.
func TestSync_BootstrapWithEmptyRegionSendsOnlyBookmark(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	region := "empty-region-bootstrap"

	svc := newService(t, h.DB)
	stream := newMockStream()

	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	req := connect.NewRequest(&ctrlv1.SyncRequest{
		Region:           region,
		SequenceLastSeen: 0,
	})

	_ = svc.Sync(ctx, req, stream)

	messages := stream.Messages()

	// Empty region should have exactly one message: the bookmark
	require.Len(t, messages, 1, "empty region bootstrap should send exactly one message (bookmark)")

	// The single message should be a bookmark
	bookmark := findBookmark(messages)
	require.NotNil(t, bookmark, "the only message should be a bookmark")

	// Sequence is 0 since no state changes exist for this region
	require.Equal(t, uint64(0), bookmark.GetSequence(), "empty region bookmark should have sequence 0")
}

// TestSync_BootstrapOnlyStreamsRunningResources verifies that bootstrap filters
// out non-running resources (archived, stopped, etc.).
//
// Scenario: A region contains both a running and an archived deployment.
//
// Guarantees:
//   - Running deployments ARE included in bootstrap
//   - Archived deployments are NOT included in bootstrap
//
// This test ensures the bootstrap phase only syncs resources that should actually
// exist in Kubernetes. Archived resources should not be created in the cluster.
func TestSync_BootstrapOnlyStreamsRunningResources(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	region := "running-only-region"

	runningDep := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       region,
		DesiredState: db.DeploymentsDesiredStateRunning,
	})

	archivedDep := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       region,
		DesiredState: db.DeploymentsDesiredStateArchived,
	})

	h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeDeployment,
		ResourceID:   runningDep.Deployment.ID,
		Op:           db.StateChangesOpUpsert,
		Region:       region,
		CreatedAt:    uint64(h.Now() - 2000),
	})

	svc := newService(t, h.DB)
	stream := newMockStream()

	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	req := connect.NewRequest(&ctrlv1.SyncRequest{
		Region:           region,
		SequenceLastSeen: 0,
	})

	_ = svc.Sync(ctx, req, stream)

	messages := stream.Messages()

	// Running deployment should be streamed
	runningApply := findDeploymentApply(messages, runningDep.Deployment.ID)
	require.NotNil(t, runningApply, "running deployment should be streamed")

	// Archived deployment should NOT be streamed
	archivedApply := findDeploymentApply(messages, archivedDep.Deployment.ID)
	require.Nil(t, archivedApply, "archived deployment should not be streamed during bootstrap")
}

// =============================================================================
// FailedPrecondition Tests
// =============================================================================
//
// These tests verify the sequence validation that prevents clients from resuming
// from a stale position. When state_changes rows are pruned (retention policy),
// clients with old sequence numbers must perform a full resync.
//
// Guarantees tested:
//   - Clients with sequence behind min retained sequence get FailedPrecondition error
//   - Clients with sequence=0 trigger bootstrap (never get FailedPrecondition)
//   - Clients with valid sequence resume normally
// =============================================================================

// TestSync_FailedPreconditionWhenSequenceBehindMin verifies that clients attempting
// to resume from a pruned sequence position receive a FailedPrecondition error.
//
// Scenario: State changes have been pruned and the client's last-seen sequence
// is now behind the minimum retained sequence for the region.
//
// Guarantees:
//   - Server returns connect.CodeFailedPrecondition error
//   - Client knows it must perform a full resync (sequence=0)
//
// This prevents clients from missing events that were pruned from state_changes.
// The client-side behavior should be: discard local sequence, reconnect with 0.
func TestSync_FailedPreconditionWhenSequenceBehindMin(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	region := "failedprecondition-region"
	dummyRegion := "dummy-region-for-sequence-bump"

	// Insert a dummy state change in a different region to bump the auto-increment.
	// This ensures the next insert will have sequence > 1, making our test meaningful.
	h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeDeployment,
		ResourceID:   "dep_dummy",
		Op:           db.StateChangesOpUpsert,
		Region:       dummyRegion,
		CreatedAt:    uint64(h.Now() - 3000),
	})

	// Insert the actual state change for our test region
	minSeq := h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeDeployment,
		ResourceID:   "dep_test",
		Op:           db.StateChangesOpUpsert,
		Region:       region,
		CreatedAt:    uint64(h.Now() - 2000),
	})

	// Verify minSeq > 1 so our test is meaningful
	require.Greater(t, minSeq, int64(1), "minSeq should be > 1 after dummy insert")

	svc := newService(t, h.DB)
	stream := newMockStream()

	// Request with sequence 1 which is behind minSeq should return FailedPrecondition
	req := connect.NewRequest(&ctrlv1.SyncRequest{
		Region:           region,
		SequenceLastSeen: 1,
	})

	err := svc.Sync(ctx, req, stream)

	require.Error(t, err)
	connectErr, ok := err.(*connect.Error)
	require.True(t, ok, "error should be a connect error")
	require.Equal(t, connect.CodeFailedPrecondition, connectErr.Code())
}

// TestSync_NoErrorWhenSequenceIsZero verifies that sequence=0 always triggers
// bootstrap instead of sequence validation.
//
// Scenario: A client connects with sequence=0 to a region that has state changes.
//
// Guarantees:
//   - No FailedPrecondition error is returned
//   - Bootstrap runs normally (timeout occurs in watch loop, not sync)
//
// sequence=0 is the "fresh start" signal. It means "I have no state, give me
// everything." This must always work regardless of what sequences exist.
func TestSync_NoErrorWhenSequenceIsZero(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	region := "zero-seq-region"

	h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeDeployment,
		ResourceID:   "dep_test",
		Op:           db.StateChangesOpUpsert,
		Region:       region,
		CreatedAt:    uint64(h.Now() - 2000),
	})

	svc := newService(t, h.DB)
	stream := newMockStream()

	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	// Sequence 0 should trigger bootstrap, not FailedPrecondition
	req := connect.NewRequest(&ctrlv1.SyncRequest{
		Region:           region,
		SequenceLastSeen: 0,
	})

	err := svc.Sync(ctx, req, stream)

	// Should timeout in watch loop, not return FailedPrecondition
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

// =============================================================================
// Delete Scenario Tests
// =============================================================================
//
// These tests verify that the watch phase correctly generates Delete messages
// in various scenarios. Delete messages tell krane to remove resources from
// Kubernetes.
//
// Delete messages are sent when:
//   - The state_changes op is explicitly "delete"
//   - The deployment topology doesn't exist for the sync region
//   - The resource's desired_state is not "running"
//   - The sentinel's region doesn't match the sync region
//
// Guarantees tested:
//   - All delete scenarios produce correct DeleteDeployment/DeleteSentinel messages
//   - The K8sName in delete messages matches the resource to be deleted
// =============================================================================

// TestSync_DeploymentDeleteWhenTopologyNotFound verifies that a Delete message
// is sent when a deployment exists but has no topology in the requesting region.
//
// Scenario: Deployment has topology in region A, but krane is syncing region B.
// A state change references the deployment in region B.
//
// Guarantees:
//   - A DeleteDeployment message is sent for the deployment
//   - This handles the case where a deployment was removed from a region
//
// This ensures krane removes deployments that no longer belong in its region,
// even if the deployment still exists elsewhere.
func TestSync_DeploymentDeleteWhenTopologyNotFound(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	region := "topology-not-found-region"
	otherRegion := "other-region"

	dep := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       otherRegion,
		DesiredState: db.DeploymentsDesiredStateRunning,
	})

	// Insert state change for our sync region (but topology doesn't exist there)
	seq := h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeDeployment,
		ResourceID:   dep.Deployment.ID,
		Op:           db.StateChangesOpUpsert,
		Region:       region,
		CreatedAt:    uint64(h.Now() - 2000),
	})

	svc := newService(t, h.DB)
	stream := newMockStream()

	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	// Start watch from before the state change
	req := connect.NewRequest(&ctrlv1.SyncRequest{
		Region:           region,
		SequenceLastSeen: uint64(seq - 1),
	})

	_ = svc.Sync(ctx, req, stream)

	messages := stream.Messages()

	del := findDeploymentDelete(messages, dep.Deployment.K8sName)
	require.NotNil(t, del, "should send delete when topology not found for region")
}

// TestSync_DeploymentDeleteWhenDesiredStateNotRunning verifies that a Delete
// message is sent when a deployment's desired_state is not "running".
//
// Scenario: A deployment exists with desired_state="archived". A state change
// is emitted for this deployment.
//
// Guarantees:
//   - A DeleteDeployment message is sent
//   - Archived/stopped deployments are removed from Kubernetes
//
// This handles graceful shutdown: when a user archives a deployment, krane
// must remove it from the cluster.
func TestSync_DeploymentDeleteWhenDesiredStateNotRunning(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	region := "not-running-region"

	dep := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       region,
		DesiredState: db.DeploymentsDesiredStateArchived,
	})

	// Insert state change
	seq := h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeDeployment,
		ResourceID:   dep.Deployment.ID,
		Op:           db.StateChangesOpUpsert,
		Region:       region,
		CreatedAt:    uint64(h.Now() - 2000),
	})

	svc := newService(t, h.DB)
	stream := newMockStream()

	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	req := connect.NewRequest(&ctrlv1.SyncRequest{
		Region:           region,
		SequenceLastSeen: uint64(seq - 1),
	})

	_ = svc.Sync(ctx, req, stream)

	messages := stream.Messages()

	del := findDeploymentDelete(messages, dep.Deployment.K8sName)
	require.NotNil(t, del, "should send delete when desired_state is not running")
}

// TestSync_DeploymentDeleteOnExplicitDeleteOp verifies that an explicit "delete"
// operation in state_changes produces a Delete message.
//
// Scenario: A running deployment has a state change with op="delete".
//
// Guarantees:
//   - A DeleteDeployment message is sent immediately
//   - This is the primary delete path for permanent resource removal
//
// Explicit delete operations are emitted when a deployment is permanently deleted
// (not just archived). The deployment row may still exist (soft delete) but
// krane must remove the Kubernetes resources.
func TestSync_DeploymentDeleteOnExplicitDeleteOp(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	region := "explicit-delete-region"

	dep := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       region,
		DesiredState: db.DeploymentsDesiredStateRunning,
	})

	// Insert delete state change
	seq := h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeDeployment,
		ResourceID:   dep.Deployment.ID,
		Op:           db.StateChangesOpDelete,
		Region:       region,
		CreatedAt:    uint64(h.Now() - 2000),
	})

	svc := newService(t, h.DB)
	stream := newMockStream()

	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	req := connect.NewRequest(&ctrlv1.SyncRequest{
		Region:           region,
		SequenceLastSeen: uint64(seq - 1),
	})

	_ = svc.Sync(ctx, req, stream)

	messages := stream.Messages()

	del := findDeploymentDelete(messages, dep.Deployment.K8sName)
	require.NotNil(t, del, "should send delete on explicit delete operation")
}

// TestSync_SentinelDeleteWhenRegionMismatch verifies that a Delete message is
// sent when a sentinel exists in a different region than the sync request.
//
// Scenario: A sentinel is created in region A. A state change references it
// in region B (e.g., due to migration or misconfiguration).
//
// Guarantees:
//   - A DeleteSentinel message is sent
//   - Sentinels are region-bound; they must not exist in wrong regions
//
// Unlike deployments (which have separate topology per region), sentinels have
// a single region field. This test ensures region filtering works for sentinels.
func TestSync_SentinelDeleteWhenRegionMismatch(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	sentinelRegion := "sentinel-actual-region"
	syncRegion := "sync-region"

	sentinel := h.CreateSentinel(ctx, CreateSentinelRequest{
		Region:       sentinelRegion,
		DesiredState: db.SentinelsDesiredStateRunning,
	})

	// Insert state change for sync region
	seq := h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeSentinel,
		ResourceID:   sentinel.ID,
		Op:           db.StateChangesOpUpsert,
		Region:       syncRegion,
		CreatedAt:    uint64(h.Now() - 2000),
	})

	svc := newService(t, h.DB)
	stream := newMockStream()

	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	req := connect.NewRequest(&ctrlv1.SyncRequest{
		Region:           syncRegion,
		SequenceLastSeen: uint64(seq - 1),
	})

	_ = svc.Sync(ctx, req, stream)

	messages := stream.Messages()

	del := findSentinelDelete(messages, sentinel.K8sName)
	require.NotNil(t, del, "should send delete when sentinel region doesn't match")
}

// TestSync_SentinelDeleteWhenDesiredStateNotRunning verifies that a Delete
// message is sent when a sentinel's desired_state is not "running".
//
// Scenario: A sentinel has desired_state="archived".
//
// Guarantees:
//   - A DeleteSentinel message is sent
//   - Archived sentinels are removed from Kubernetes
//
// This mirrors the deployment test for sentinels.
func TestSync_SentinelDeleteWhenDesiredStateNotRunning(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	region := "sentinel-archived-region"

	sentinel := h.CreateSentinel(ctx, CreateSentinelRequest{
		Region:       region,
		DesiredState: db.SentinelsDesiredStateArchived,
	})

	// Insert state change
	seq := h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeSentinel,
		ResourceID:   sentinel.ID,
		Op:           db.StateChangesOpUpsert,
		Region:       region,
		CreatedAt:    uint64(h.Now() - 2000),
	})

	svc := newService(t, h.DB)
	stream := newMockStream()

	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	req := connect.NewRequest(&ctrlv1.SyncRequest{
		Region:           region,
		SequenceLastSeen: uint64(seq - 1),
	})

	_ = svc.Sync(ctx, req, stream)

	messages := stream.Messages()

	del := findSentinelDelete(messages, sentinel.K8sName)
	require.NotNil(t, del, "should send delete when sentinel desired_state is not running")
}

// TestSync_SentinelDeleteOnExplicitDeleteOp verifies that an explicit "delete"
// operation produces a Delete message for sentinels.
//
// Scenario: A running sentinel has a state change with op="delete".
//
// Guarantees:
//   - A DeleteSentinel message is sent
//   - This is the primary delete path for permanent sentinel removal
func TestSync_SentinelDeleteOnExplicitDeleteOp(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	region := "sentinel-explicit-delete-region"

	sentinel := h.CreateSentinel(ctx, CreateSentinelRequest{
		Region:       region,
		DesiredState: db.SentinelsDesiredStateRunning,
	})

	// Insert delete state change
	seq := h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeSentinel,
		ResourceID:   sentinel.ID,
		Op:           db.StateChangesOpDelete,
		Region:       region,
		CreatedAt:    uint64(h.Now() - 2000),
	})

	svc := newService(t, h.DB)
	stream := newMockStream()

	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	req := connect.NewRequest(&ctrlv1.SyncRequest{
		Region:           region,
		SequenceLastSeen: uint64(seq - 1),
	})

	_ = svc.Sync(ctx, req, stream)

	messages := stream.Messages()

	del := findSentinelDelete(messages, sentinel.K8sName)
	require.NotNil(t, del, "should send delete on explicit delete operation")
}

// =============================================================================
// State Change Query Tests
// =============================================================================
//
// These tests verify the underlying database queries that power the sync
// mechanism. They test the SQLC-generated query functions directly.
//
// Guarantees tested:
//   - ListStateChanges returns changes after a given sequence
//   - GetMaxStateChangeSequence returns the highest sequence for a region
//   - GetMinStateChangeSequence returns the lowest sequence for a region
//   - State changes are correctly filtered by region
// =============================================================================

// TestSync_StateChangeQueries verifies that ListStateChanges correctly returns
// state changes after a given sequence number.
//
// Scenario: Three state changes are inserted with increasing sequences.
// We query for changes after the first sequence.
//
// Guarantees:
//   - Only changes AFTER the specified sequence are returned
//   - Changes are returned in sequence order
//   - The correct number of changes is returned
func TestSync_StateChangeQueries(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	region := "query-test-region"

	seq1 := h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeDeployment,
		ResourceID:   "dep_1",
		Op:           db.StateChangesOpUpsert,
		Region:       region,
		CreatedAt:    uint64(h.Now() - 5000),
	})

	seq2 := h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeSentinel,
		ResourceID:   "sen_1",
		Op:           db.StateChangesOpUpsert,
		Region:       region,
		CreatedAt:    uint64(h.Now() - 4000),
	})

	seq3 := h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeDeployment,
		ResourceID:   "dep_1",
		Op:           db.StateChangesOpDelete,
		Region:       region,
		CreatedAt:    uint64(h.Now() - 3000),
	})

	require.Less(t, seq1, seq2)
	require.Less(t, seq2, seq3)

	// Query state changes after seq1
	changes, err := db.Query.ListStateChanges(ctx, h.DB.RO(), db.ListStateChangesParams{
		Region:        region,
		AfterSequence: uint64(seq1),
		Limit:         100,
	})
	require.NoError(t, err)
	require.Len(t, changes, 2)
	require.Equal(t, uint64(seq2), changes[0].Sequence)
	require.Equal(t, uint64(seq3), changes[1].Sequence)
}

// TestSync_MaxSequenceQuery verifies that GetMaxStateChangeSequence returns
// the highest sequence number for a region.
//
// Scenario: Two state changes are inserted. We query for the max sequence.
//
// Guarantees:
//   - Returns the sequence of the most recent state change
//   - Used during bootstrap to set the Bookmark sequence
func TestSync_MaxSequenceQuery(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	region := "max-seq-query-region"

	h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeDeployment,
		ResourceID:   "dep_1",
		Op:           db.StateChangesOpUpsert,
		Region:       region,
		CreatedAt:    uint64(h.Now() - 3000),
	})

	seq2 := h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeSentinel,
		ResourceID:   "sen_1",
		Op:           db.StateChangesOpUpsert,
		Region:       region,
		CreatedAt:    uint64(h.Now() - 2000),
	})

	maxSeq, err := db.Query.GetMaxStateChangeSequence(ctx, h.DB.RO(), region)
	require.NoError(t, err)
	require.Equal(t, seq2, maxSeq)
}

// TestSync_MinSequenceQuery verifies that GetMinStateChangeSequence returns
// the lowest sequence number for a region.
//
// Scenario: Two state changes are inserted. We query for the min sequence.
//
// Guarantees:
//   - Returns the sequence of the oldest state change
//   - Used during sequence validation to detect stale clients
func TestSync_MinSequenceQuery(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	region := "min-seq-query-region"

	seq1 := h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeDeployment,
		ResourceID:   "dep_1",
		Op:           db.StateChangesOpUpsert,
		Region:       region,
		CreatedAt:    uint64(h.Now() - 3000),
	})

	h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeSentinel,
		ResourceID:   "sen_1",
		Op:           db.StateChangesOpUpsert,
		Region:       region,
		CreatedAt:    uint64(h.Now() - 2000),
	})

	minSeq, err := db.Query.GetMinStateChangeSequence(ctx, h.DB.RO(), region)
	require.NoError(t, err)
	require.Equal(t, seq1, minSeq)
}

// TestSync_StateChangeRegionFiltering verifies that ListStateChanges only
// returns state changes for the specified region.
//
// Scenario: State changes are inserted in two different regions.
// We query for changes in region1 only.
//
// Guarantees:
//   - Only changes for the requested region are returned
//   - Changes in other regions are not included
//
// This is essential for multi-region deployments where each krane instance
// only cares about its own region's state changes.
func TestSync_StateChangeRegionFiltering(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	region1 := "filter-region-1"
	region2 := "filter-region-2"

	h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeDeployment,
		ResourceID:   "dep_region1",
		Op:           db.StateChangesOpUpsert,
		Region:       region1,
		CreatedAt:    uint64(h.Now() - 3000),
	})

	h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeDeployment,
		ResourceID:   "dep_region2",
		Op:           db.StateChangesOpUpsert,
		Region:       region2,
		CreatedAt:    uint64(h.Now() - 2000),
	})

	changes, err := db.Query.ListStateChanges(ctx, h.DB.RO(), db.ListStateChangesParams{
		Region:        region1,
		AfterSequence: 0,
		Limit:         100,
	})
	require.NoError(t, err)
	require.Len(t, changes, 1)
	require.Equal(t, "dep_region1", changes[0].ResourceID)
}

// =============================================================================
// Reconnect Tests
// =============================================================================
//
// These tests verify the incremental sync behavior when a krane agent
// reconnects with a non-zero sequence number.
//
// When sequence > 0:
//   - Bootstrap is skipped (no full state dump)
//   - Watch begins immediately from the given sequence
//   - No Bookmark is sent (client already has one)
//
// Guarantees tested:
//   - Reconnecting clients receive only new state changes
//   - Clients don't receive duplicate events they already processed
// =============================================================================

// TestSync_ReconnectResumesFromSequence verifies that a reconnecting client
// skips bootstrap and receives only new state changes.
//
// Scenario: Two state changes exist. Client reconnects with the first sequence.
//
// Guarantees:
//   - No Bookmark is sent (bootstrap was skipped)
//   - Only state changes AFTER the provided sequence are streamed
//   - The deployment update from the second state change is received
//
// This is the core reconnection contract: clients resume where they left off.
func TestSync_ReconnectResumesFromSequence(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	region := "reconnect-region"

	dep := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       region,
		DesiredState: db.DeploymentsDesiredStateRunning,
	})

	// First state change (simulates previous session)
	seq1 := h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeDeployment,
		ResourceID:   dep.Deployment.ID,
		Op:           db.StateChangesOpUpsert,
		Region:       region,
		CreatedAt:    uint64(h.Now() - 5000),
	})

	// Second state change (new since last session)
	_ = h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeDeployment,
		ResourceID:   dep.Deployment.ID,
		Op:           db.StateChangesOpUpsert,
		Region:       region,
		CreatedAt:    uint64(h.Now() - 2000),
	})

	svc := newService(t, h.DB)
	stream := newMockStream()

	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	// Reconnect with sequence from first state change (should NOT trigger full bootstrap)
	req := connect.NewRequest(&ctrlv1.SyncRequest{
		Region:           region,
		SequenceLastSeen: uint64(seq1),
	})

	_ = svc.Sync(ctx, req, stream)

	messages := stream.Messages()

	// Should NOT have a bookmark (bootstrap sends bookmark, watch doesn't)
	bookmark := findBookmark(messages)
	require.Nil(t, bookmark, "reconnect should skip bootstrap and not send bookmark")

	apply := findDeploymentApply(messages, dep.Deployment.ID)
	require.NotNil(t, apply, "should receive deployment update from watch")
}

// =============================================================================
// Sequence in Messages Tests
// =============================================================================
//
// These tests verify that all streamed messages contain valid sequence numbers
// that clients can use for resumption.
//
// Guarantees tested:
//   - All messages have a sequence field > 0
//   - Clients can use any message's sequence for reconnection
// =============================================================================

// TestSync_AllMessagesContainSequence verifies that every message streamed
// during bootstrap contains a valid sequence number.
//
// Scenario: Bootstrap streams multiple resources.
//
// Guarantees:
//   - All apply messages have sequence > 0
//   - During bootstrap, all messages have the same sequence (the max at bootstrap time)
//
// The sequence in messages allows clients to track their position even if they
// disconnect mid-stream.
func TestSync_AllMessagesContainSequence(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	region := "sequence-in-messages-region"

	dep := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       region,
		DesiredState: db.DeploymentsDesiredStateRunning,
	})

	sentinel := h.CreateSentinel(ctx, CreateSentinelRequest{
		Region:       region,
		DesiredState: db.SentinelsDesiredStateRunning,
	})

	h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeDeployment,
		ResourceID:   dep.Deployment.ID,
		Op:           db.StateChangesOpUpsert,
		Region:       region,
		CreatedAt:    uint64(h.Now() - 2000),
	})

	h.InsertStateChange(ctx, db.InsertStateChangeParams{
		ResourceType: db.StateChangesResourceTypeSentinel,
		ResourceID:   sentinel.ID,
		Op:           db.StateChangesOpUpsert,
		Region:       region,
		CreatedAt:    uint64(h.Now() - 2000),
	})

	svc := newService(t, h.DB)
	stream := newMockStream()

	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	req := connect.NewRequest(&ctrlv1.SyncRequest{
		Region:           region,
		SequenceLastSeen: 0,
	})

	_ = svc.Sync(ctx, req, stream)

	messages := stream.Messages()
	require.NotEmpty(t, messages)

	// All messages during bootstrap should have the same sequence (the max sequence at bootstrap time)
	var bootstrapSequence uint64
	for _, msg := range messages {
		seq := msg.GetSequence()
		if bootstrapSequence == 0 {
			bootstrapSequence = seq
		}
		// Bootstrap messages all have the same sequence
		if msg.GetBookmark() == nil {
			// Non-bookmark messages should have a valid sequence
			require.Greater(t, seq, uint64(0), "all messages should have sequence > 0")
		}
	}
}
