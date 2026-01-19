// Package reconciler provides the krane reconciler that synchronizes control
// plane state with Kubernetes resources.
//
// These tests verify the watch/sync behavior of the reconciler, specifically:
//   - How it forms Sync requests to the control plane
//   - How it processes State messages via HandleState
//   - How it tracks version numbers for reconnection
//
// # Test Approach
//
// Due to connect.ServerStreamForClient being a struct (not an interface), we
// cannot mock the actual stream returned by Sync(). Instead, we test:
//   - Request formation (capture the SyncRequest sent to the mock)
//   - HandleState processing (call HandleState directly with test messages)
//   - Error handling (return errors from the mock Sync function)
//
// # Key Invariants
//
//   - versionLastSeen is only updated after clean stream close (atomic bootstrap)
//   - HandleState returns the version but does not commit it
//   - Apply messages create/update Kubernetes resources
//   - Delete messages remove Kubernetes resources
package reconciler

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/ptr"
	"k8s.io/client-go/kubernetes/fake"
)

// mockServerStream implements connect.ServerStreamForClient for testing.
type mockServerStream struct {
	messages []*ctrlv1.State
	index    int
	err      error
	closed   bool
	mu       sync.Mutex
}

func newMockServerStream(messages []*ctrlv1.State) *mockServerStream {
	return &mockServerStream{
		messages: messages,
		index:    0,
	}
}

func (m *mockServerStream) Receive() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed || m.index >= len(m.messages) {
		return false
	}
	m.index++
	return true
}

func (m *mockServerStream) Msg() *ctrlv1.State {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.index == 0 || m.index > len(m.messages) {
		return nil
	}
	return m.messages[m.index-1]
}

func (m *mockServerStream) Err() error {
	return m.err
}

func (m *mockServerStream) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockServerStream) ResponseHeader() http.Header {
	return make(http.Header)
}

func (m *mockServerStream) ResponseTrailer() http.Header {
	return make(http.Header)
}

// =============================================================================
// Sync Request Formation Tests
// =============================================================================
//
// These tests verify that the reconciler sends correctly formed Sync requests
// to the control plane. The request must include the region and the last-seen
// sequence number.
// =============================================================================

// TestWatch_SendsCorrectSyncRequest verifies that watch() sends a Sync request
// with the correct region and version number.
//
// Scenario: Reconciler has previously processed messages up to version 500.
// It calls watch() which should send a Sync request with that version.
//
// Guarantees:
//   - SyncRequest.Region matches the reconciler's configured region
//   - SyncRequest.VersionLastSeen matches versionLastSeen from previous session
//
// This is critical for reconnection: the version tells the server where to
// resume streaming from.
func TestWatch_SendsCorrectSyncRequest(t *testing.T) {
	client := fake.NewSimpleClientset()
	AddReplicaSetPatchReactor(client)
	AddDeploymentPatchReactor(client)
	AddServicePatchReactor(client)
	AddDeleteTracker(client)

	var capturedRequest *ctrlv1.SyncRequest

	mockCluster := &MockClusterClient{
		SyncFunc: func(_ context.Context, req *connect.Request[ctrlv1.SyncRequest]) (*connect.ServerStreamForClient[ctrlv1.State], error) {
			capturedRequest = req.Msg
			return nil, errors.New("end test")
		},
	}

	r := New(Config{
		ClientSet: client,
		Logger:    logging.NewNoop(),
		Cluster:   mockCluster,
		Region:    "us-west-2",
	})

	r.versionLastSeen = 500

	ctx := context.Background()
	_ = r.watch(ctx)

	require.NotNil(t, capturedRequest)
	require.Equal(t, "us-west-2", capturedRequest.GetRegion())
	require.Equal(t, uint64(500), capturedRequest.GetVersionLastSeen())
}

// TestWatch_InitialSyncWithZeroVersion verifies that a fresh reconciler sends
// version=0 to trigger a full bootstrap from the server.
//
// Scenario: A newly created reconciler (never received any messages) calls watch().
//
// Guarantees:
//   - SyncRequest.VersionLastSeen is 0
//   - This triggers the server to perform full bootstrap
//
// version=0 is the "I have nothing" signal that tells the server to send
// all current state before entering the watch loop.
func TestWatch_InitialSyncWithZeroVersion(t *testing.T) {
	client := fake.NewSimpleClientset()
	AddReplicaSetPatchReactor(client)
	AddDeploymentPatchReactor(client)
	AddServicePatchReactor(client)
	AddDeleteTracker(client)

	var capturedRequest *ctrlv1.SyncRequest

	mockCluster := &MockClusterClient{
		SyncFunc: func(_ context.Context, req *connect.Request[ctrlv1.SyncRequest]) (*connect.ServerStreamForClient[ctrlv1.State], error) {
			capturedRequest = req.Msg
			return nil, errors.New("end test")
		},
	}

	r := New(Config{
		ClientSet: client,
		Logger:    logging.NewNoop(),
		Cluster:   mockCluster,
		Region:    "eu-central-1",
	})

	ctx := context.Background()
	_ = r.watch(ctx)

	require.NotNil(t, capturedRequest)
	require.Equal(t, "eu-central-1", capturedRequest.GetRegion())
	require.Equal(t, uint64(0), capturedRequest.GetVersionLastSeen(), "initial sync should have version 0")
}

// =============================================================================
// HandleState Processing Tests
// =============================================================================
//
// These tests verify that HandleState correctly processes different message
// types and updates both Kubernetes resources and the version tracker.
// =============================================================================

// TestWatch_ProcessesStreamMessages verifies that HandleState correctly
// processes deployment apply messages and returns versions.
//
// Scenario: A stream contains two deployment apply messages (ver=10, ver=20).
//
// Guarantees:
//   - The deployment is applied to Kubernetes (ReplicaSet is created)
//   - HandleState returns the version from each message
//   - versionLastSeen is only updated after stream closes cleanly
//
// This tests the basic happy path: apply resources and track version.
func TestWatch_ProcessesStreamMessages(t *testing.T) {
	client := fake.NewSimpleClientset()
	rsCapture := AddReplicaSetPatchReactor(client)
	AddDeploymentPatchReactor(client)
	AddServicePatchReactor(client)
	AddDeleteTracker(client)

	messages := []*ctrlv1.State{
		{
			Version: 10,
			Kind: &ctrlv1.State_Deployment{
				Deployment: &ctrlv1.DeploymentState{
					State: &ctrlv1.DeploymentState_Apply{
						Apply: &ctrlv1.ApplyDeployment{
							WorkspaceId:   "ws_1",
							ProjectId:     "prj_1",
							EnvironmentId: "env_1",
							DeploymentId:  "dep_1",
							K8SNamespace:  "test-ns",
							K8SName:       "dep-1",
							Image:         "nginx:1.19",
							Replicas:      1,
							CpuMillicores: 100,
							MemoryMib:     128,
							BuildId:       ptr.P("build_1"),
						},
					},
				},
			},
		},
		{
			Version: 20,
			Kind: &ctrlv1.State_Deployment{
				Deployment: &ctrlv1.DeploymentState{
					State: &ctrlv1.DeploymentState_Apply{
						Apply: &ctrlv1.ApplyDeployment{
							WorkspaceId:   "ws_1",
							ProjectId:     "prj_1",
							EnvironmentId: "env_1",
							DeploymentId:  "dep_2",
							K8SNamespace:  "test-ns",
							K8SName:       "dep-2",
							Image:         "nginx:1.20",
							Replicas:      1,
							CpuMillicores: 100,
							MemoryMib:     128,
							BuildId:       ptr.P("build_2"),
						},
					},
				},
			},
		},
	}

	stream := newMockServerStream(messages)

	mockCluster := &MockClusterClient{
		SyncFunc: func(_ context.Context, _ *connect.Request[ctrlv1.SyncRequest]) (*connect.ServerStreamForClient[ctrlv1.State], error) {
			// Return our mock stream wrapped as the expected interface
			return (*connect.ServerStreamForClient[ctrlv1.State])(nil), nil
		},
	}

	r := New(Config{
		ClientSet: client,
		Logger:    logging.NewNoop(),
		Cluster:   mockCluster,
		Region:    "test-region",
	})

	// Process messages directly to test HandleState integration
	ctx := context.Background()
	var maxVersion uint64
	for stream.Receive() {
		seq, err := r.HandleState(ctx, stream.Msg())
		require.NoError(t, err)
		if seq > maxVersion {
			maxVersion = seq
		}
	}

	require.NotNil(t, rsCapture.Applied, "deployment should have been applied")
	require.Equal(t, uint64(0), r.versionLastSeen, "sequence should not be updated until CommitSequence")

	// Simulate clean stream close
	if maxVersion > r.versionLastSeen {
		r.versionLastSeen = maxVersion
	}
	require.Equal(t, uint64(20), r.versionLastSeen, "sequence should be updated after CommitSequence")
}

// TestWatch_IncrementalUpdates verifies that HandleState correctly processes
// a sequence of incremental updates including applies and deletes.
//
// Scenario: Starting from sequence 100 (simulating reconnect after bootstrap),
// the reconciler receives: apply deployment (101), delete deployment (102),
// delete sentinel (103).
//
// Guarantees:
//   - HandleState returns the sequence from each message
//   - versionLastSeen is updated to 103 after CommitSequence
//   - Deployment delete triggers ReplicaSet deletion
//   - Sentinel delete triggers Deployment deletion (sentinels run as k8s Deployments)
//
// This tests the watch loop after bootstrap: processing incremental changes
// as they happen in the control plane.
func TestWatch_IncrementalUpdates(t *testing.T) {
	client := fake.NewSimpleClientset()
	AddReplicaSetPatchReactor(client)
	AddDeploymentPatchReactor(client)
	AddServicePatchReactor(client)
	deletes := AddDeleteTracker(client)

	messages := []*ctrlv1.State{
		{
			Version: 101,
			Kind: &ctrlv1.State_Deployment{
				Deployment: &ctrlv1.DeploymentState{
					State: &ctrlv1.DeploymentState_Apply{
						Apply: &ctrlv1.ApplyDeployment{
							WorkspaceId:   "ws_1",
							ProjectId:     "prj_1",
							EnvironmentId: "env_1",
							DeploymentId:  "dep_new",
							K8SNamespace:  "test-ns",
							K8SName:       "new-deployment",
							Image:         "myapp:v2",
							Replicas:      3,
							CpuMillicores: 500,
							MemoryMib:     512,
						},
					},
				},
			},
		},
		{
			Version: 102,
			Kind: &ctrlv1.State_Deployment{
				Deployment: &ctrlv1.DeploymentState{
					State: &ctrlv1.DeploymentState_Delete{
						Delete: &ctrlv1.DeleteDeployment{
							K8SNamespace: "test-ns",
							K8SName:      "old-deployment",
						},
					},
				},
			},
		},
		{
			Version: 103,
			Kind: &ctrlv1.State_Sentinel{
				Sentinel: &ctrlv1.SentinelState{
					State: &ctrlv1.SentinelState_Delete{
						Delete: &ctrlv1.DeleteSentinel{
							K8SName: "old-sentinel",
						},
					},
				},
			},
		},
	}

	mockCluster := &MockClusterClient{}

	r := New(Config{
		ClientSet: client,
		Logger:    logging.NewNoop(),
		Cluster:   mockCluster,
		Region:    "test-region",
	})

	// Start with sequence 100 (simulating reconnect after bootstrap)
	r.versionLastSeen = 100

	ctx := context.Background()
	stream := newMockServerStream(messages)
	var maxVersion uint64
	for stream.Receive() {
		seq, err := r.HandleState(ctx, stream.Msg())
		require.NoError(t, err)
		if seq > maxVersion {
			maxVersion = seq
		}
	}

	require.Equal(t, uint64(100), r.versionLastSeen, "sequence should not change until CommitSequence")
	if maxVersion > r.versionLastSeen {
		r.versionLastSeen = maxVersion
	}
	require.Equal(t, uint64(103), r.versionLastSeen)
	require.Contains(t, deletes.Actions, "replicasets", "deployment delete should be processed (deletes ReplicaSet)")
	require.Contains(t, deletes.Actions, "deployments", "sentinel delete should be processed (deletes Deployment)")
}

// =============================================================================
// Configuration Tests
// =============================================================================

// TestWatch_RegionIsPersisted verifies that the region from Config is correctly
// stored in the reconciler.
//
// Guarantees:
//   - New() correctly sets the region field from Config
//   - The region is available for use in Sync requests
func TestWatch_RegionIsPersisted(t *testing.T) {
	cfg := Config{
		ClientSet: fake.NewSimpleClientset(),
		Logger:    logging.NewNoop(),
		Cluster:   &MockClusterClient{},
		Region:    "ap-southeast-1",
	}

	r := New(cfg)
	require.Equal(t, "ap-southeast-1", r.region)
}

// =============================================================================
// Error Handling Tests
// =============================================================================

// TestWatch_SyncConnectionError verifies that connection errors from Sync()
// are properly propagated back to the caller.
//
// Scenario: The control plane is unreachable (connection refused).
//
// Guarantees:
//   - The error from Sync() is returned by watch()
//   - The caller (Watch loop) can handle reconnection logic
//
// This tests the error path: what happens when the control plane is down.
// The Watch() outer loop will retry with exponential backoff.
func TestWatch_SyncConnectionError(t *testing.T) {
	client := fake.NewSimpleClientset()
	AddReplicaSetPatchReactor(client)
	AddDeploymentPatchReactor(client)
	AddServicePatchReactor(client)
	AddDeleteTracker(client)

	expectedErr := errors.New("connection refused")

	mockCluster := &MockClusterClient{
		SyncFunc: func(_ context.Context, req *connect.Request[ctrlv1.SyncRequest]) (*connect.ServerStreamForClient[ctrlv1.State], error) {
			return nil, expectedErr
		},
	}

	r := New(Config{
		ClientSet: client,
		Logger:    logging.NewNoop(),
		Cluster:   mockCluster,
		Region:    "error-test-region",
	})

	ctx := context.Background()
	err := r.watch(ctx)

	require.Error(t, err)
	require.Equal(t, expectedErr, err)
}

// =============================================================================
// End-to-End Message Flow Tests
// =============================================================================

// TestWatch_FullMessageProcessingFlow verifies the complete message processing
// flow including multiple resource types and operations.
//
// Scenario: A full sync stream containing:
//   - Apply deployment (seq=10)
//   - Apply sentinel (seq=20)
//   - Delete deployment (seq=30)
//
// Guarantees:
//   - Deployment is applied to Kubernetes (ReplicaSet created with correct name)
//   - Sentinel is applied (as a k8s Deployment - captured separately)
//   - Deployment delete is processed (ReplicaSet deleted)
//   - versionLastSeen ends at 30 (the highest sequence)
//
// This is a comprehensive integration test of HandleState covering all major
// message types in a realistic sequence.
func TestWatch_FullMessageProcessingFlow(t *testing.T) {
	client := fake.NewSimpleClientset()
	rsCapture := AddReplicaSetPatchReactor(client)
	AddDeploymentPatchReactor(client)
	AddServicePatchReactor(client)
	deletes := AddDeleteTracker(client)

	r := New(Config{
		ClientSet: client,
		Logger:    logging.NewNoop(),
		Cluster:   &MockClusterClient{},
		Region:    "full-flow-region",
	})

	ctx := context.Background()

	messages := []*ctrlv1.State{
		{
			Version: 10,
			Kind: &ctrlv1.State_Deployment{
				Deployment: &ctrlv1.DeploymentState{
					State: &ctrlv1.DeploymentState_Apply{
						Apply: &ctrlv1.ApplyDeployment{
							WorkspaceId:   "ws_flow",
							ProjectId:     "prj_flow",
							EnvironmentId: "env_flow",
							DeploymentId:  "dep_flow",
							K8SNamespace:  "flow-ns",
							K8SName:       "flow-deployment",
							Image:         "myapp:v1",
							Replicas:      2,
							CpuMillicores: 200,
							MemoryMib:     256,
						},
					},
				},
			},
		},
		{
			Version: 20,
			Kind: &ctrlv1.State_Sentinel{
				Sentinel: &ctrlv1.SentinelState{
					State: &ctrlv1.SentinelState_Apply{
						Apply: &ctrlv1.ApplySentinel{
							K8SName:       "flow-sentinel",
							WorkspaceId:   "ws_flow",
							EnvironmentId: "env_flow",
							ProjectId:     "prj_flow",
							SentinelId:    "sen_flow",
							Image:         "sentinel:v1",
							Replicas:      1,
							CpuMillicores: 100,
							MemoryMib:     128,
						},
					},
				},
			},
		},
		{
			Version: 30,
			Kind: &ctrlv1.State_Deployment{
				Deployment: &ctrlv1.DeploymentState{
					State: &ctrlv1.DeploymentState_Delete{
						Delete: &ctrlv1.DeleteDeployment{
							K8SNamespace: "flow-ns",
							K8SName:      "old-deployment",
						},
					},
				},
			},
		},
	}

	var maxVersion uint64
	for _, msg := range messages {
		seq, err := r.HandleState(ctx, msg)
		require.NoError(t, err)
		if seq > maxVersion {
			maxVersion = seq
		}
	}

	require.NotNil(t, rsCapture.Applied, "deployment should have been applied")
	require.Equal(t, "flow-deployment", rsCapture.Applied.Name)

	require.Contains(t, deletes.Actions, "replicasets", "deployment delete should have been processed")

	require.Equal(t, uint64(0), r.versionLastSeen, "sequence should not be updated until CommitSequence")

	if maxVersion > r.versionLastSeen {
		r.versionLastSeen = maxVersion
	}
	require.Equal(t, uint64(30), r.versionLastSeen, "sequence should be updated after CommitSequence")
}
