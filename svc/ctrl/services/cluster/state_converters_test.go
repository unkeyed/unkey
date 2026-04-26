package cluster

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
)

func TestDeploymentRowToState_Running(t *testing.T) {
	row := deploymentRow{
		dt: db.DeploymentTopology{
			DesiredStatus:          db.DeploymentTopologyDesiredStatusRunning,
			AutoscalingReplicasMin: 1,
			AutoscalingReplicasMax: 3,
		},
		d: db.Deployment{
			ID:             "deploy_123",
			K8sName:        "my-app",
			WorkspaceID:    "ws_1",
			ProjectID:      "prj_1",
			EnvironmentID:  "env_1",
			AppID:          "app_1",
			Image:          sql.NullString{Valid: true, String: "registry.io/app:v1"},
			CpuMillicores:  250,
			MemoryMib:      256,
			Port:           8080,
			ShutdownSignal: db.DeploymentsShutdownSignalSIGTERM,
		},
		k8sNamespace:    sql.NullString{Valid: true, String: "ws-namespace"},
		environmentSlug: "production",
		regionName:      "us-east-1",
	}

	state, err := deploymentRowToState(row, 42)
	require.NoError(t, err)
	require.NotNil(t, state)

	require.Equal(t, uint64(42), state.GetVersion())

	apply := state.GetApply()
	require.NotNil(t, apply, "running status should produce an ApplyDeployment")
	require.Equal(t, "deploy_123", apply.GetDeploymentId())
	require.Equal(t, "my-app", apply.GetK8SName())
	require.Equal(t, "ws-namespace", apply.GetK8SNamespace())
	require.Equal(t, int64(250), apply.GetCpuMillicores())
	require.Equal(t, uint32(1), apply.GetAutoscaling().GetMinReplicas())
	require.Equal(t, uint32(3), apply.GetAutoscaling().GetMaxReplicas())
}

func TestDeploymentRowToState_Stopped(t *testing.T) {
	row := deploymentRow{
		dt: db.DeploymentTopology{
			DesiredStatus: db.DeploymentTopologyDesiredStatusStopped,
		},
		d: db.Deployment{
			K8sName: "my-app",
		},
		k8sNamespace: sql.NullString{Valid: true, String: "ws-namespace"},
	}

	state, err := deploymentRowToState(row, 7)
	require.NoError(t, err)
	require.NotNil(t, state)

	require.Equal(t, uint64(7), state.GetVersion())

	del := state.GetDelete()
	require.NotNil(t, del, "stopped status should produce a DeleteDeployment")
	require.Equal(t, "my-app", del.GetK8SName())
	require.Equal(t, "ws-namespace", del.GetK8SNamespace())
}

func TestSentinelToState_Running(t *testing.T) {
	sentinel := db.Sentinel{
		ID:              "snt_123",
		K8sName:         "sentinel-abc",
		WorkspaceID:     "ws_1",
		ProjectID:       "prj_1",
		EnvironmentID:   "env_1",
		DesiredState:    db.SentinelsDesiredStateRunning,
		Image:           "registry.io/sentinel:v1",
		DesiredReplicas: 3,
		CpuMillicores:   100,
		MemoryMib:       128,
	}

	state := sentinelToState(sentinel, 10)
	require.NotNil(t, state)
	require.Equal(t, uint64(10), state.GetVersion())

	apply := state.GetApply()
	require.NotNil(t, apply, "running state should produce an ApplySentinel")
	require.Equal(t, "snt_123", apply.GetSentinelId())
	require.Equal(t, int32(3), apply.GetReplicas())
}

func TestSentinelToState_Archived(t *testing.T) {
	sentinel := db.Sentinel{
		K8sName:      "sentinel-abc",
		DesiredState: db.SentinelsDesiredStateArchived,
	}

	state := sentinelToState(sentinel, 5)
	require.NotNil(t, state)

	del := state.GetDelete()
	require.NotNil(t, del, "archived state should produce a DeleteSentinel")
	require.Equal(t, "sentinel-abc", del.GetK8SName())
}

func TestSentinelToState_Standby(t *testing.T) {
	sentinel := db.Sentinel{
		K8sName:      "sentinel-abc",
		DesiredState: db.SentinelsDesiredStateStandby,
	}

	state := sentinelToState(sentinel, 5)
	require.NotNil(t, state)

	del := state.GetDelete()
	require.NotNil(t, del, "standby state should produce a DeleteSentinel")
}

func TestCiliumPolicyToState(t *testing.T) {
	policy := db.CiliumNetworkPolicy{
		ID:           "cnp_123",
		K8sName:      "sentinel-ingress-to-my-app",
		K8sNamespace: "ws-namespace",
		Policy:       []byte(`{"spec": {}}`),
	}

	state := ciliumPolicyToState(policy, 99)
	require.Equal(t, uint64(99), state.GetVersion())

	apply := state.GetApply()
	require.NotNil(t, apply)
	require.Equal(t, "cnp_123", apply.GetCiliumNetworkPolicyId())
	require.Equal(t, "sentinel-ingress-to-my-app", apply.GetK8SName())
	require.Equal(t, "ws-namespace", apply.GetK8SNamespace())
	require.Equal(t, []byte(`{"spec": {}}`), apply.GetPolicy())
}
