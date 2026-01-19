package reconciler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/ptr"
)

// Tests for version tracking behavior.
//
// The reconciler uses a two-phase commit for version tracking:
// 1. HandleState processes state and returns the version (but doesn't commit it)
// 2. versionLastSeen is updated only after clean stream close
//
// This ensures atomic bootstrap: if a stream breaks mid-bootstrap, the client
// retries from version 0 rather than skipping resources that were never received.

func TestHandleState_ReturnsVersion(t *testing.T) {
	ctx := context.Background()
	h := NewTestHarness(t)
	r := h.Reconciler

	require.Equal(t, uint64(0), r.versionLastSeen, "initial version should be 0")

	state := &ctrlv1.State{
		Version: 42,
		Kind: &ctrlv1.State_Deployment{
			Deployment: &ctrlv1.DeploymentState{
				State: &ctrlv1.DeploymentState_Apply{
					Apply: &ctrlv1.ApplyDeployment{
						WorkspaceId:   "ws_123",
						ProjectId:     "prj_123",
						EnvironmentId: "env_123",
						DeploymentId:  "dep_123",
						K8SNamespace:  "test-namespace",
						K8SName:       "test-deployment",
						Image:         "nginx:1.19",
						Replicas:      1,
						CpuMillicores: 100,
						MemoryMib:     128,
						BuildId:       ptr.P("build_123"),
					},
				},
			},
		},
	}

	ver, err := r.HandleState(ctx, state)
	require.NoError(t, err)
	require.Equal(t, uint64(42), ver, "should return state version")
	require.Equal(t, uint64(0), r.versionLastSeen, "versionLastSeen should not change until stream closes cleanly")
}

func TestHandleState_DoesNotCommitVersion(t *testing.T) {
	ctx := context.Background()
	h := NewTestHarness(t)
	r := h.Reconciler

	states := []*ctrlv1.State{
		{
			Version: 100,
			Kind: &ctrlv1.State_Deployment{
				Deployment: &ctrlv1.DeploymentState{
					State: &ctrlv1.DeploymentState_Delete{
						Delete: &ctrlv1.DeleteDeployment{
							K8SNamespace: "test-namespace",
							K8SName:      "test-deployment",
						},
					},
				},
			},
		},
		{
			Version: 200,
			Kind: &ctrlv1.State_Sentinel{
				Sentinel: &ctrlv1.SentinelState{
					State: &ctrlv1.SentinelState_Apply{
						Apply: &ctrlv1.ApplySentinel{
							WorkspaceId:   "ws_123",
							ProjectId:     "prj_123",
							EnvironmentId: "env_123",
							SentinelId:    "sentinel_123",
							K8SName:       "test-sentinel",
							Image:         "sentinel:1.0",
							Replicas:      2,
							CpuMillicores: 100,
							MemoryMib:     128,
						},
					},
				},
			},
		},
		{
			Version: 300,
			Kind: &ctrlv1.State_Sentinel{
				Sentinel: &ctrlv1.SentinelState{
					State: &ctrlv1.SentinelState_Delete{
						Delete: &ctrlv1.DeleteSentinel{
							K8SName: "test-sentinel",
						},
					},
				},
			},
		},
	}

	for _, state := range states {
		_, err := r.HandleState(ctx, state)
		require.NoError(t, err)
	}

	require.Equal(t, uint64(0), r.versionLastSeen, "versionLastSeen should remain 0 until stream closes cleanly")
}
