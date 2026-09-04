package environment_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
)

func TestPromoteDeploymentSwapsLiveRoutes(t *testing.T) {
	f := newFixture(t)

	// The target carries a parked stop from an earlier demotion; the untouched
	// deployment carries one too and proves the parked stop mechanism works.
	f.scheduleStop(t, f.candidate.ID)
	f.scheduleStop(t, f.other.ID)

	require.NoError(t, f.promote(f.env.ID, f.candidate.ID))

	f.requireLive(t, f.candidate.ID, false)
	f.requireRoutes(t, f.candidate.ID, f.live.ID)
	f.requireDesiredStateAfterStopDelay(t, f.candidate.ID, mysqltype.DeploymentsDesiredStateRunning)
	f.requireDesiredStateAfterStopDelay(t, f.other.ID, mysqltype.DeploymentsDesiredStateStopped)
	require.Equal(t, 1, countAudits(t, f.ctx, f.db, f.workspaceID, auditlog.DeploymentPromoteEvent, f.candidate.ID, f.actorID))
}

func TestPromoteDeploymentConfirmsRollback(t *testing.T) {
	f := newFixture(t)
	f.moveStickyRoutes(t, f.candidate.ID)
	f.setLive(t, f.candidate.ID, true)

	require.NoError(t, f.promote(f.env.ID, f.candidate.ID))

	f.requireLive(t, f.candidate.ID, false)
	f.requireRoutes(t, f.candidate.ID, f.live.ID)
	require.Equal(t, 1, countAudits(t, f.ctx, f.db, f.workspaceID, auditlog.DeploymentPromoteEvent, f.candidate.ID, f.actorID))
}

func TestPromoteDeploymentRejectsForeignEnvironment(t *testing.T) {
	f := newFixture(t)
	foreign := f.newProductionEnvironment(t, "production-kebap")

	err := f.promote(foreign.ID, f.candidate.ID)
	require.ErrorContains(t, err, "keyed environment")

	f.requireLive(t, f.live.ID, false)
	f.requireRoutes(t, f.live.ID, f.live.ID)
	require.Equal(t, 0, countAudits(t, f.ctx, f.db, f.workspaceID, auditlog.DeploymentPromoteEvent, f.candidate.ID, f.actorID))
}
