package cluster

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestDeploymentRowToState_Running(t *testing.T) {
	row := db.FindDeploymentTopologyByDeploymentAndRegionRow{
		DesiredStatus:          db.DeploymentTopologyDesiredStatusRunning,
		AutoscalingReplicasMin: 1,
		AutoscalingReplicasMax: 3,
		ID:                     "deploy_123",
		K8sName:                "my-app",
		WorkspaceID:            "ws_1",
		ProjectID:              "prj_1",
		EnvironmentID:          "env_1",
		AppID:                  "app_1",
		ImageResolved:          sql.NullString{Valid: true, String: "registry.io/app:v1"},
		CpuMillicores:          250,
		MemoryMib:              256,
		Port:                   8080,
		ShutdownSignal:         db.DeploymentsShutdownSignalSIGTERM,
		K8sNamespace:           "ws-namespace",
		EnvironmentSlug:        "production",
		RegionName:             "us-east-1",
	}

	state, err := deploymentRowToState(row)
	require.NoError(t, err)
	require.NotNil(t, state)

	apply := state.GetApply()
	require.NotNil(t, apply, "running status should produce an ApplyDeployment")
	require.Equal(t, "deploy_123", apply.GetDeploymentId())
	require.Equal(t, "my-app", apply.GetK8SName())
	require.Equal(t, "ws-namespace", apply.GetK8SNamespace())
	require.Equal(t, "registry.io/app:v1", apply.GetImage())
	require.Equal(t, int64(250), apply.GetCpuMillicores())
	require.Equal(t, uint32(1), apply.GetAutoscaling().GetMinReplicas())
	require.Equal(t, uint32(3), apply.GetAutoscaling().GetMaxReplicas())
}

func TestDeploymentRowToState_Stopped(t *testing.T) {
	row := db.FindDeploymentTopologyByDeploymentAndRegionRow{
		DesiredStatus: db.DeploymentTopologyDesiredStatusStopped,
		K8sName:       "my-app",
		K8sNamespace:  "ws-namespace",
	}

	state, err := deploymentRowToState(row)
	require.NoError(t, err)
	require.NotNil(t, state)

	del := state.GetDelete()
	require.NotNil(t, del, "stopped status should produce a DeleteDeployment")
	require.Equal(t, "my-app", del.GetK8SName())
	require.Equal(t, "ws-namespace", del.GetK8SNamespace())
}

// TestDeploymentRowToState_PrivateNetworkReplicaHost guarantees that only
// deployments of enrolled workspaces receive a replica host, which is what
// makes Krane point their Pods at undns, and that the host is the app slug.
func TestDeploymentRowToState_PrivateNetworkReplicaHost(t *testing.T) {
	for _, tt := range []struct {
		name     string
		enrolled bool
		slug     string
		want     string
	}{
		{name: "enrolled", enrolled: true, slug: "api", want: "api.unkey.internal"},
		{name: "not_enrolled", enrolled: false, slug: "api", want: ""},
		{name: "enrolled_invalid_slug", enrolled: true, slug: "Bad_Slug", want: ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			state, err := deploymentRowToState(db.ListAllDeploymentTopologiesByRegionRow{
				TopologyDesiredStatus:  db.DeploymentTopologyDesiredStatusRunning,
				DeploymentID:           "deploy_123",
				AppSlug:                tt.slug,
				PrivateNetworkEnrolled: tt.enrolled,
			})
			require.NoError(t, err)
			require.Equal(t, tt.want, state.GetApply().GetPrivateNetworkReplicaHost())
		})
	}
}
