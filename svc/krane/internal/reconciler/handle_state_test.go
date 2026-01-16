package reconciler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/ptr"
)

// These tests verify HandleState correctly routes to the appropriate handler.
// Detailed behavior of each handler is tested in their respective test files.

func TestHandleState_DeploymentApply(t *testing.T) {
	ctx := context.Background()
	h := NewTestHarness(t)
	r := h.Reconciler

	state := &ctrlv1.State{
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
						Replicas:      3,
						CpuMillicores: 100,
						MemoryMib:     128,
						BuildId:       ptr.P("build_123"),
					},
				},
			},
		},
	}

	err := r.HandleState(ctx, state)
	require.NoError(t, err)
	require.NotNil(t, h.ReplicaSets.Applied, "should route to ApplyDeployment")
}

func TestHandleState_DeploymentDelete(t *testing.T) {
	ctx := context.Background()
	h := NewTestHarness(t)
	r := h.Reconciler

	state := &ctrlv1.State{
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
	}

	err := r.HandleState(ctx, state)
	require.NoError(t, err)
	require.Contains(t, h.Deletes.Actions, "replicasets", "should route to DeleteDeployment")
}

func TestHandleState_SentinelApply(t *testing.T) {
	ctx := context.Background()
	h := NewTestHarness(t)
	r := h.Reconciler

	state := &ctrlv1.State{
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
	}

	err := r.HandleState(ctx, state)
	require.NoError(t, err)
	require.NotNil(t, h.Deployments.Applied, "should route to ApplySentinel (Deployment)")
	require.NotNil(t, h.Services.Applied, "should route to ApplySentinel (Service)")
}

func TestHandleState_SentinelDelete(t *testing.T) {
	ctx := context.Background()
	h := NewTestHarness(t)
	r := h.Reconciler

	state := &ctrlv1.State{
		Kind: &ctrlv1.State_Sentinel{
			Sentinel: &ctrlv1.SentinelState{
				State: &ctrlv1.SentinelState_Delete{
					Delete: &ctrlv1.DeleteSentinel{
						K8SName: "test-sentinel",
					},
				},
			},
		},
	}

	err := r.HandleState(ctx, state)
	require.NoError(t, err)
	require.Contains(t, h.Deletes.Actions, "services", "should route to DeleteSentinel (Service)")
	require.Contains(t, h.Deletes.Actions, "deployments", "should route to DeleteSentinel (Deployment)")
}

func TestHandleState_UnknownStateType(t *testing.T) {
	ctx := context.Background()
	h := NewTestHarness(t)
	r := h.Reconciler

	state := &ctrlv1.State{
		Kind: nil,
	}

	err := r.HandleState(ctx, state)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown state type")
}

func TestHandleState_NilState(t *testing.T) {
	ctx := context.Background()
	h := NewTestHarness(t)
	r := h.Reconciler

	err := r.HandleState(ctx, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "state is nil")
}
