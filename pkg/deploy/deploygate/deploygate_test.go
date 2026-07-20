package deploygate

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
)

func promoteBase() PromoteInput {
	return PromoteInput{
		Status:              db.DeploymentsStatusReady,
		DesiredState:        db.DeploymentsDesiredStateRunning,
		EnvironmentSlug:     envProduction,
		CurrentDeploymentID: "dep_live",
		DeploymentID:        "dep_target",
		IsRolledBack:        false,
	}
}

func rollbackBase() RollbackInput {
	return RollbackInput{
		Status:              db.DeploymentsStatusReady,
		DesiredState:        db.DeploymentsDesiredStateRunning,
		EnvironmentSlug:     envProduction,
		CurrentDeploymentID: "dep_live",
		DeploymentID:        "dep_target",
	}
}

func TestCheckPromoteTarget(t *testing.T) {
	t.Run("eligible", func(t *testing.T) {
		require.Equal(t, TargetOK, CheckPromoteTarget(promoteBase()))
	})

	t.Run("reason order", func(t *testing.T) {
		in := promoteBase()
		in.Status = "building"
		require.Equal(t, TargetNotReady, CheckPromoteTarget(in))

		in = promoteBase()
		in.DesiredState = "stopped"
		require.Equal(t, TargetDraining, CheckPromoteTarget(in))

		in = promoteBase()
		in.EnvironmentSlug = "preview"
		require.Equal(t, TargetNotProduction, CheckPromoteTarget(in))

		in = promoteBase()
		in.CurrentDeploymentID = ""
		require.Equal(t, TargetNoCurrentDeployment, CheckPromoteTarget(in))
	})

	t.Run("already live is rejected", func(t *testing.T) {
		in := promoteBase()
		in.CurrentDeploymentID = in.DeploymentID
		require.Equal(t, TargetAlreadyCurrent, CheckPromoteTarget(in))
	})

	t.Run("promoting the current deployment while rolled back is allowed", func(t *testing.T) {
		in := promoteBase()
		in.CurrentDeploymentID = in.DeploymentID
		in.IsRolledBack = true
		require.Equal(t, TargetOK, CheckPromoteTarget(in), "confirming a rollback")
	})
}

func TestCheckRollbackTarget(t *testing.T) {
	t.Run("eligible", func(t *testing.T) {
		require.Equal(t, TargetOK, CheckRollbackTarget(rollbackBase()))
	})

	t.Run("rolling back to the current deployment is rejected", func(t *testing.T) {
		in := rollbackBase()
		in.CurrentDeploymentID = in.DeploymentID
		require.Equal(t, TargetAlreadyCurrent, CheckRollbackTarget(in))
	})

	t.Run("shares the core preconditions", func(t *testing.T) {
		in := rollbackBase()
		in.Status = "failed"
		require.Equal(t, TargetNotReady, CheckRollbackTarget(in))
	})
}

func TestCheckStopTarget(t *testing.T) {
	running := StopInput{Status: db.DeploymentsStatusReady, DesiredState: db.DeploymentsDesiredStateRunning, EnvironmentSlug: "preview"}

	require.Equal(t, StopOK, CheckStopTarget(running))

	notReady := running
	notReady.Status = db.DeploymentsStatusStopped
	require.Equal(t, StopNotRunning, CheckStopTarget(notReady))

	draining := running
	draining.DesiredState = db.DeploymentsDesiredStateStopped
	require.Equal(t, StopAlreadyStopping, CheckStopTarget(draining))

	prod := running
	prod.EnvironmentSlug = envProduction
	require.Equal(t, StopIsProduction, CheckStopTarget(prod))
}

func TestCheckStartTarget(t *testing.T) {
	// A stopped deployment is keyed on desired_state, not status: it may still be
	// draining (status ready) while its intent is stopped.
	stopped := StartInput{DesiredState: db.DeploymentsDesiredStateStopped, EnvironmentSlug: "preview"}

	require.Equal(t, StartOK, CheckStartTarget(stopped), "wakeable while draining toward stopped")

	notStopped := stopped
	notStopped.DesiredState = db.DeploymentsDesiredStateRunning
	require.Equal(t, StartNotStopped, CheckStartTarget(notStopped))

	prod := stopped
	prod.EnvironmentSlug = envProduction
	require.Equal(t, StartIsProduction, CheckStartTarget(prod))

	suspended := stopped
	suspended.SpendSuspended = true
	require.Equal(t, StartSpendSuspended, CheckStartTarget(suspended))
}
