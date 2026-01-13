package reconciler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func TestHandleState_ApplyDeployment_UpdatesStateWithinTimeout(t *testing.T) {
	h := NewTestHarness(t)
	ctx := context.Background()

	var called atomic.Bool
	h.ControlPlane.UpdateDeploymentStateFunc = func(_ context.Context, req *connect.Request[ctrlv1.UpdateDeploymentStateRequest]) (*connect.Response[ctrlv1.UpdateDeploymentStateResponse], error) {
		called.Store(true)
		return connect.NewResponse(&ctrlv1.UpdateDeploymentStateResponse{}), nil
	}

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

	go func() {
		err := h.Reconciler.HandleState(ctx, state)
		require.NoError(t, err)
	}()

	require.Eventually(t, called.Load, 10*time.Second, 100*time.Millisecond,
		"expected UpdateDeploymentState to be called within 10 seconds")
}

func TestHandleState_ApplySentinel_UpdatesStateWithinTimeout(t *testing.T) {
	h := NewTestHarness(t)
	ctx := context.Background()

	var called atomic.Bool
	h.ControlPlane.UpdateSentinelStateFunc = func(_ context.Context, req *connect.Request[ctrlv1.UpdateSentinelStateRequest]) (*connect.Response[ctrlv1.UpdateSentinelStateResponse], error) {
		called.Store(true)
		return connect.NewResponse(&ctrlv1.UpdateSentinelStateResponse{}), nil
	}

	state := &ctrlv1.State{
		Kind: &ctrlv1.State_Sentinel{
			Sentinel: &ctrlv1.SentinelState{
				State: &ctrlv1.SentinelState_Apply{
					Apply: &ctrlv1.ApplySentinel{
						WorkspaceId:   "ws_123",
						ProjectId:     "prj_123",
						EnvironmentId: "env_123",
						SentinelId:    "sentinel_123",
						K8SNamespace:  "test-namespace",
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

	go func() {
		err := h.Reconciler.HandleState(ctx, state)
		require.NoError(t, err)
	}()

	require.Eventually(t, called.Load, 10*time.Second, 100*time.Millisecond,
		"expected UpdateSentinelState to be called within 10 seconds")
}

func TestHandleState_DeleteDeployment_UpdatesStateWithinTimeout(t *testing.T) {
	h := NewTestHarness(t)
	ctx := context.Background()

	var called atomic.Bool
	h.ControlPlane.UpdateDeploymentStateFunc = func(_ context.Context, req *connect.Request[ctrlv1.UpdateDeploymentStateRequest]) (*connect.Response[ctrlv1.UpdateDeploymentStateResponse], error) {
		called.Store(true)
		return connect.NewResponse(&ctrlv1.UpdateDeploymentStateResponse{}), nil
	}

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

	go func() {
		err := h.Reconciler.HandleState(ctx, state)
		require.NoError(t, err)
	}()

	require.Eventually(t, called.Load, 10*time.Second, 100*time.Millisecond,
		"expected UpdateDeploymentState to be called within 10 seconds")
}

func TestHandleState_DeleteSentinel_UpdatesStateWithinTimeout(t *testing.T) {
	h := NewTestHarness(t)
	ctx := context.Background()

	called := atomic.Bool{}
	h.ControlPlane.UpdateSentinelStateFunc = func(_ context.Context, req *connect.Request[ctrlv1.UpdateSentinelStateRequest]) (*connect.Response[ctrlv1.UpdateSentinelStateResponse], error) {
		called.Store(true)
		return connect.NewResponse(&ctrlv1.UpdateSentinelStateResponse{}), nil
	}

	state := &ctrlv1.State{
		Kind: &ctrlv1.State_Sentinel{
			Sentinel: &ctrlv1.SentinelState{
				State: &ctrlv1.SentinelState_Delete{
					Delete: &ctrlv1.DeleteSentinel{
						K8SNamespace: "test-namespace",
						K8SName:      "test-sentinel",
					},
				},
			},
		},
	}

	go func() {
		err := h.Reconciler.HandleState(ctx, state)
		require.NoError(t, err)
	}()

	require.Eventually(t, called.Load, 10*time.Second, 100*time.Millisecond,
		"expected UpdateSentinelState to be called within 10 seconds")
}

func TestHandleState_ApplyDeployment_StateUpdateContainsCorrectData(t *testing.T) {
	h := NewTestHarness(t)
	ctx := context.Background()

	var receivedUpdate atomic.Pointer[ctrlv1.UpdateDeploymentStateRequest]
	h.ControlPlane.UpdateDeploymentStateFunc = func(_ context.Context, req *connect.Request[ctrlv1.UpdateDeploymentStateRequest]) (*connect.Response[ctrlv1.UpdateDeploymentStateResponse], error) {
		receivedUpdate.Store(req.Msg)
		return connect.NewResponse(&ctrlv1.UpdateDeploymentStateResponse{}), nil
	}

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

	go func() {
		err := h.Reconciler.HandleState(ctx, state)
		require.NoError(t, err)
	}()

	require.Eventually(t, func() bool { return receivedUpdate.Load() != nil }, 10*time.Second, 100*time.Millisecond,
		"expected UpdateDeploymentState to be called within 10 seconds")

	update := receivedUpdate.Load()
	require.NotNil(t, update.GetUpdate(), "expected an update change type")
	require.Equal(t, "test-deployment", update.GetUpdate().GetK8SName())
}

func TestHandleState_DeleteDeployment_StateUpdateContainsCorrectData(t *testing.T) {
	h := NewTestHarness(t)
	ctx := context.Background()

	var receivedUpdate atomic.Pointer[ctrlv1.UpdateDeploymentStateRequest]
	h.ControlPlane.UpdateDeploymentStateFunc = func(_ context.Context, req *connect.Request[ctrlv1.UpdateDeploymentStateRequest]) (*connect.Response[ctrlv1.UpdateDeploymentStateResponse], error) {
		receivedUpdate.Store(req.Msg)
		return connect.NewResponse(&ctrlv1.UpdateDeploymentStateResponse{}), nil
	}

	state := &ctrlv1.State{
		Kind: &ctrlv1.State_Deployment{
			Deployment: &ctrlv1.DeploymentState{
				State: &ctrlv1.DeploymentState_Delete{
					Delete: &ctrlv1.DeleteDeployment{
						K8SNamespace: "test-namespace",
						K8SName:      "test-deployment-to-delete",
					},
				},
			},
		},
	}

	go func() {
		err := h.Reconciler.HandleState(ctx, state)
		require.NoError(t, err)
	}()

	require.Eventually(t, func() bool { return receivedUpdate.Load() != nil }, 10*time.Second, 100*time.Millisecond,
		"expected UpdateDeploymentState to be called within 10 seconds")

	update := receivedUpdate.Load()
	require.NotNil(t, update.GetDelete(), "expected a delete change type")
	require.Equal(t, "test-deployment-to-delete", update.GetDelete().GetK8SName())
}

func TestHandleState_ApplySentinel_StateUpdateContainsCorrectData(t *testing.T) {
	h := NewTestHarness(t)
	ctx := context.Background()

	var receivedUpdate atomic.Pointer[ctrlv1.UpdateSentinelStateRequest]
	h.ControlPlane.UpdateSentinelStateFunc = func(_ context.Context, req *connect.Request[ctrlv1.UpdateSentinelStateRequest]) (*connect.Response[ctrlv1.UpdateSentinelStateResponse], error) {
		receivedUpdate.Store(req.Msg)
		return connect.NewResponse(&ctrlv1.UpdateSentinelStateResponse{}), nil
	}

	state := &ctrlv1.State{
		Kind: &ctrlv1.State_Sentinel{
			Sentinel: &ctrlv1.SentinelState{
				State: &ctrlv1.SentinelState_Apply{
					Apply: &ctrlv1.ApplySentinel{
						WorkspaceId:   "ws_123",
						ProjectId:     "prj_123",
						EnvironmentId: "env_123",
						SentinelId:    "sentinel_123",
						K8SNamespace:  "test-namespace",
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

	go func() {
		err := h.Reconciler.HandleState(ctx, state)
		require.NoError(t, err)
	}()

	require.Eventually(t, func() bool { return receivedUpdate.Load() != nil }, 10*time.Second, 100*time.Millisecond,
		"expected UpdateSentinelState to be called within 10 seconds")

	update := receivedUpdate.Load()
	require.Equal(t, "test-sentinel", update.GetK8SName())
}

func TestHandleState_DeleteSentinel_StateUpdateContainsCorrectData(t *testing.T) {
	h := NewTestHarness(t)
	ctx := context.Background()

	var receivedUpdate atomic.Pointer[ctrlv1.UpdateSentinelStateRequest]
	h.ControlPlane.UpdateSentinelStateFunc = func(_ context.Context, req *connect.Request[ctrlv1.UpdateSentinelStateRequest]) (*connect.Response[ctrlv1.UpdateSentinelStateResponse], error) {
		receivedUpdate.Store(req.Msg)
		return connect.NewResponse(&ctrlv1.UpdateSentinelStateResponse{}), nil
	}

	state := &ctrlv1.State{
		Kind: &ctrlv1.State_Sentinel{
			Sentinel: &ctrlv1.SentinelState{
				State: &ctrlv1.SentinelState_Delete{
					Delete: &ctrlv1.DeleteSentinel{
						K8SNamespace: "test-namespace",
						K8SName:      "test-sentinel-to-delete",
					},
				},
			},
		},
	}

	go func() {
		err := h.Reconciler.HandleState(ctx, state)
		require.NoError(t, err)
	}()

	require.Eventually(t, func() bool { return receivedUpdate.Load() != nil }, 10*time.Second, 100*time.Millisecond,
		"expected UpdateSentinelState to be called within 10 seconds")

	update := receivedUpdate.Load()
	require.Equal(t, "test-sentinel-to-delete", update.GetK8SName())
	require.Equal(t, int32(0), update.GetAvailableReplicas())
}
