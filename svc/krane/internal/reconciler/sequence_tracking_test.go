package reconciler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/ptr"
)

// Tests for sequence tracking behavior in HandleState.
// The reconciler tracks sequenceLastSeen to resume from the correct position on reconnect.

func TestHandleState_UpdatesSequenceAfterDeploymentApply(t *testing.T) {
	ctx := context.Background()
	h := NewTestHarness(t)
	r := h.Reconciler

	require.Equal(t, uint64(0), r.sequenceLastSeen, "initial sequence should be 0")

	state := &ctrlv1.State{
		Sequence: 42,
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

	err := r.HandleState(ctx, state)
	require.NoError(t, err)
	require.Equal(t, uint64(42), r.sequenceLastSeen, "sequence should be updated after apply")
}

func TestHandleState_UpdatesSequenceAfterDeploymentDelete(t *testing.T) {
	ctx := context.Background()
	h := NewTestHarness(t)
	r := h.Reconciler

	state := &ctrlv1.State{
		Sequence: 100,
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
	require.Equal(t, uint64(100), r.sequenceLastSeen)
}

func TestHandleState_UpdatesSequenceAfterSentinelApply(t *testing.T) {
	ctx := context.Background()
	h := NewTestHarness(t)
	r := h.Reconciler

	state := &ctrlv1.State{
		Sequence: 200,
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
	require.Equal(t, uint64(200), r.sequenceLastSeen)
}

func TestHandleState_UpdatesSequenceAfterSentinelDelete(t *testing.T) {
	ctx := context.Background()
	h := NewTestHarness(t)
	r := h.Reconciler

	state := &ctrlv1.State{
		Sequence: 300,
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
	require.Equal(t, uint64(300), r.sequenceLastSeen)
}

func TestHandleState_UpdatesSequenceFromBookmark(t *testing.T) {
	ctx := context.Background()
	h := NewTestHarness(t)
	r := h.Reconciler

	state := &ctrlv1.State{
		Sequence: 500, // State-level sequence
		Kind: &ctrlv1.State_Bookmark{
			Bookmark: &ctrlv1.Bookmark{
				Sequence: 999, // Bookmark-specific sequence takes precedence
			},
		},
	}

	err := r.HandleState(ctx, state)
	require.NoError(t, err)
	require.Equal(t, uint64(999), r.sequenceLastSeen, "bookmark sequence should override state sequence")
}

func TestHandleState_SequenceOnlyIncreases(t *testing.T) {
	ctx := context.Background()
	h := NewTestHarness(t)
	r := h.Reconciler

	// First event with sequence 100
	state1 := &ctrlv1.State{
		Sequence: 100,
		Kind: &ctrlv1.State_Deployment{
			Deployment: &ctrlv1.DeploymentState{
				State: &ctrlv1.DeploymentState_Delete{
					Delete: &ctrlv1.DeleteDeployment{
						K8SNamespace: "test-namespace",
						K8SName:      "test-deployment-1",
					},
				},
			},
		},
	}

	err := r.HandleState(ctx, state1)
	require.NoError(t, err)
	require.Equal(t, uint64(100), r.sequenceLastSeen)

	// Second event with lower sequence (should not decrease)
	state2 := &ctrlv1.State{
		Sequence: 50, // Lower sequence
		Kind: &ctrlv1.State_Deployment{
			Deployment: &ctrlv1.DeploymentState{
				State: &ctrlv1.DeploymentState_Delete{
					Delete: &ctrlv1.DeleteDeployment{
						K8SNamespace: "test-namespace",
						K8SName:      "test-deployment-2",
					},
				},
			},
		},
	}

	err = r.HandleState(ctx, state2)
	require.NoError(t, err)
	require.Equal(t, uint64(100), r.sequenceLastSeen, "sequence should not decrease")

	// Third event with higher sequence (should increase)
	state3 := &ctrlv1.State{
		Sequence: 150,
		Kind: &ctrlv1.State_Deployment{
			Deployment: &ctrlv1.DeploymentState{
				State: &ctrlv1.DeploymentState_Delete{
					Delete: &ctrlv1.DeleteDeployment{
						K8SNamespace: "test-namespace",
						K8SName:      "test-deployment-3",
					},
				},
			},
		},
	}

	err = r.HandleState(ctx, state3)
	require.NoError(t, err)
	require.Equal(t, uint64(150), r.sequenceLastSeen, "sequence should increase")
}

func TestHandleState_SequenceZeroDoesNotReset(t *testing.T) {
	ctx := context.Background()
	h := NewTestHarness(t)
	r := h.Reconciler

	// Set initial sequence
	state1 := &ctrlv1.State{
		Sequence: 100,
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

	err := r.HandleState(ctx, state1)
	require.NoError(t, err)
	require.Equal(t, uint64(100), r.sequenceLastSeen)

	// Event with sequence 0 should not reset
	state2 := &ctrlv1.State{
		Sequence: 0,
		Kind: &ctrlv1.State_Deployment{
			Deployment: &ctrlv1.DeploymentState{
				State: &ctrlv1.DeploymentState_Delete{
					Delete: &ctrlv1.DeleteDeployment{
						K8SNamespace: "test-namespace",
						K8SName:      "another-deployment",
					},
				},
			},
		},
	}

	err = r.HandleState(ctx, state2)
	require.NoError(t, err)
	require.Equal(t, uint64(100), r.sequenceLastSeen, "sequence should not reset to 0")
}

func TestHandleState_BootstrapSequence(t *testing.T) {
	ctx := context.Background()
	h := NewTestHarness(t)
	r := h.Reconciler

	// Simulate bootstrap: all state items have the same sequence from GetMaxStateChangeSequence
	bootstrapSequence := uint64(500)

	states := []*ctrlv1.State{
		{
			Sequence: bootstrapSequence,
			Kind: &ctrlv1.State_Deployment{
				Deployment: &ctrlv1.DeploymentState{
					State: &ctrlv1.DeploymentState_Apply{
						Apply: &ctrlv1.ApplyDeployment{
							WorkspaceId:   "ws_1",
							ProjectId:     "prj_1",
							EnvironmentId: "env_1",
							DeploymentId:  "dep_1",
							K8SNamespace:  "test-namespace",
							K8SName:       "deployment-1",
							Image:         "nginx:1.19",
							Replicas:      1,
							CpuMillicores: 100,
							MemoryMib:     128,
						},
					},
				},
			},
		},
		{
			Sequence: bootstrapSequence,
			Kind: &ctrlv1.State_Deployment{
				Deployment: &ctrlv1.DeploymentState{
					State: &ctrlv1.DeploymentState_Apply{
						Apply: &ctrlv1.ApplyDeployment{
							WorkspaceId:   "ws_2",
							ProjectId:     "prj_2",
							EnvironmentId: "env_2",
							DeploymentId:  "dep_2",
							K8SNamespace:  "test-namespace",
							K8SName:       "deployment-2",
							Image:         "nginx:1.20",
							Replicas:      2,
							CpuMillicores: 200,
							MemoryMib:     256,
						},
					},
				},
			},
		},
		{
			Sequence: bootstrapSequence,
			Kind: &ctrlv1.State_Sentinel{
				Sentinel: &ctrlv1.SentinelState{
					State: &ctrlv1.SentinelState_Apply{
						Apply: &ctrlv1.ApplySentinel{
							WorkspaceId:   "ws_3",
							ProjectId:     "prj_3",
							EnvironmentId: "env_3",
							SentinelId:    "sentinel_1",
							K8SName:       "sentinel-1",
							Image:         "sentinel:1.0",
							Replicas:      1,
							CpuMillicores: 50,
							MemoryMib:     64,
						},
					},
				},
			},
		},
		// Bookmark signals end of bootstrap
		{
			Sequence: bootstrapSequence,
			Kind: &ctrlv1.State_Bookmark{
				Bookmark: &ctrlv1.Bookmark{
					Sequence: bootstrapSequence,
				},
			},
		},
	}

	for _, state := range states {
		err := r.HandleState(ctx, state)
		require.NoError(t, err)
	}

	require.Equal(t, bootstrapSequence, r.sequenceLastSeen, "sequence should be set to bootstrap watermark")
}

func TestReconciler_InitialSequenceIsZero(t *testing.T) {
	h := NewTestHarness(t)
	require.Equal(t, uint64(0), h.Reconciler.sequenceLastSeen, "new reconciler should start with sequence 0")
}
