package readiness

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

type instancesDatabase struct {
	db.Database
	instances []db.Instance
}

func (d *instancesDatabase) FindInstancesByDeploymentId(context.Context, string) ([]db.Instance, error) {
	return d.instances, nil
}

func TestInstancesHealthy_RequiresOneRunningInstanceWhenMinimumIsZero(t *testing.T) {
	database := &instancesDatabase{}
	regionMinReplicas := map[string]uint32{"us-east-1": 0}

	healthy, err := InstancesHealthy(t.Context(), database, "deployment_1", regionMinReplicas, 1)
	require.NoError(t, err)
	require.False(t, healthy)

	database.instances = []db.Instance{{
		RegionID: "us-east-1",
		Status:   db.InstancesStatusRunning,
	}}
	healthy, err = InstancesHealthy(t.Context(), database, "deployment_1", regionMinReplicas, 1)
	require.NoError(t, err)
	require.True(t, healthy)
}
