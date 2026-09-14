package environment_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
)

func TestRollbackDeploymentSwitchesLiveRoutesBack(t *testing.T) {
	f := newFixture(t)
	f.scheduleStop(t, f.candidate.ID)

	require.NoError(t, f.rollback(f.env.ID, f.live.ID, f.candidate.ID))

	f.requireLive(t, f.candidate.ID, true)
	f.requireRoutes(t, f.candidate.ID, f.live.ID)
	f.requireDesiredStateAfterStopDelay(t, f.candidate.ID, mysqltype.DeploymentsDesiredStateRunning)
	require.Equal(t, 1, countAudits(t, f.ctx, f.db, f.workspaceID, auditlog.DeploymentRollbackEvent, f.candidate.ID, f.actorID))
}

func TestRollbackDeploymentRejectsStaleSource(t *testing.T) {
	f := newFixture(t)

	err := f.rollback(f.env.ID, f.other.ID, f.candidate.ID)
	require.ErrorContains(t, err, "no longer live")

	f.requireLive(t, f.live.ID, false)
	f.requireRoutes(t, f.live.ID, f.live.ID)
	require.Equal(t, 0, countAudits(t, f.ctx, f.db, f.workspaceID, auditlog.DeploymentRollbackEvent, f.candidate.ID, f.actorID))
}

func TestRollbackDeploymentRejectsForeignEnvironment(t *testing.T) {
	f := newFixture(t)
	foreign := f.newProductionEnvironment(t, "production-kebap")

	err := f.rollback(foreign.ID, f.live.ID, f.candidate.ID)
	require.ErrorContains(t, err, "keyed environment")

	f.requireLive(t, f.live.ID, false)
	f.requireRoutes(t, f.live.ID, f.live.ID)
	require.Equal(t, 0, countAudits(t, f.ctx, f.db, f.workspaceID, auditlog.DeploymentRollbackEvent, f.candidate.ID, f.actorID))
}
