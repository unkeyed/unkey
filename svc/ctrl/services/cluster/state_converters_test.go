package cluster

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestDeploymentRowToState_Running(t *testing.T) {
	row := deploymentRow{
		desiredStatus:          db.DeploymentTopologyDesiredStatusRunning,
		autoscalingReplicasMin: 1,
		autoscalingReplicasMax: 3,
		deploymentID:           "deploy_123",
		k8sName:                "my-app",
		workspaceID:            "ws_1",
		projectID:              "prj_1",
		environmentID:          "env_1",
		appID:                  "app_1",
		image:                  sql.NullString{Valid: true, String: "registry.io/app:v1"},
		cpuMillicores:          250,
		memoryMiB:              256,
		port:                   8080,
		shutdownSignal:         db.DeploymentsShutdownSignalSIGTERM,
		k8sNamespace:           sql.NullString{Valid: true, String: "ws-namespace"},
		environmentSlug:        "production",
		regionName:             "us-east-1",
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
		desiredStatus: db.DeploymentTopologyDesiredStatusStopped,
		k8sName:       "my-app",
		k8sNamespace:  sql.NullString{Valid: true, String: "ws-namespace"},
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
