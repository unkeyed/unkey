package cluster

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestDeploymentRowToState_Running(t *testing.T) {
	row := db.FindDeploymentTopologyByDeploymentAndRegionRow{
		DesiredStatus:          db.DeploymentTopologyDesiredStatusRunning,
		AutoscalingReplicasMin: 1,
		AutoscalingReplicasMax: 3,
		ID:                     uid.New(uid.DeploymentPrefix),
		K8sName:                "my-app",
		WorkspaceID:            uid.New(uid.WorkspacePrefix),
		ProjectID:              uid.New(uid.ProjectPrefix),
		EnvironmentID:          uid.New(uid.EnvironmentPrefix),
		AppID:                  uid.New(uid.AppPrefix),
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
	require.Equal(t, row.ID, apply.GetDeploymentId())
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

func TestDeploymentRowToState_PrivateNetworking(t *testing.T) {
	for _, enrolled := range []bool{true, false} {
		t.Run(fmt.Sprintf("enrolled=%t", enrolled), func(t *testing.T) {
			state, err := deploymentRowToState(db.ListAllDeploymentTopologiesByRegionRow{
				TopologyDesiredStatus:  db.DeploymentTopologyDesiredStatusRunning,
				DeploymentID:           uid.New(uid.DeploymentPrefix),
				DeploymentCapabilities: mysqltype.DeploymentCapabilities{PrivateNetworking: enrolled},
			})
			require.NoError(t, err)
			require.NotNil(t, state.GetApply().GetCapabilities(), "Ctrl always sends the saved capabilities")
			require.Equal(t, enrolled, state.GetApply().GetCapabilities().GetPrivateNetworking())
		})
	}
}
