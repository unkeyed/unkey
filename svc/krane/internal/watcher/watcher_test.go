package watcher

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
)

func TestDispatch_NilEvent(t *testing.T) {
	w := &Watcher{}
	err := w.dispatch(context.Background(), &ctrlv1.DeploymentChangeEvent{
		Version: 1,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil event")
}

func TestDispatch_NilDeploymentState(t *testing.T) {
	w := &Watcher{}
	err := w.dispatch(context.Background(), &ctrlv1.DeploymentChangeEvent{
		Version: 1,
		Event: &ctrlv1.DeploymentChangeEvent_Deployment{
			Deployment: nil,
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil deployment state")
}

func TestDispatch_NilSentinelState(t *testing.T) {
	w := &Watcher{}
	err := w.dispatch(context.Background(), &ctrlv1.DeploymentChangeEvent{
		Version: 1,
		Event: &ctrlv1.DeploymentChangeEvent_Sentinel{
			Sentinel: nil,
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil sentinel state")
}

func TestDispatch_NilCiliumPolicyState(t *testing.T) {
	w := &Watcher{}
	err := w.dispatch(context.Background(), &ctrlv1.DeploymentChangeEvent{
		Version: 1,
		Event: &ctrlv1.DeploymentChangeEvent_CiliumNetworkPolicy{
			CiliumNetworkPolicy: nil,
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil policy state")
}
