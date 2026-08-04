package deploygate

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	dbtype "github.com/unkeyed/unkey/pkg/mysql/types"
)

func promoteBase() PromoteInput {
	return PromoteInput{
		Status:              dbtype.DeploymentsStatusReady,
		DesiredState:        dbtype.DeploymentsDesiredStateRunning,
		EnvironmentKind:     dbtype.EnvironmentKindProduction,
		CurrentDeploymentID: "dep_live",
		DeploymentID:        "dep_target",
		IsRolledBack:        false,
	}
}

func rollbackBase() RollbackInput {
	return RollbackInput{
		Status:              dbtype.DeploymentsStatusReady,
		DesiredState:        dbtype.DeploymentsDesiredStateRunning,
		EnvironmentKind:     dbtype.EnvironmentKindProduction,
		CurrentDeploymentID: "dep_live",
		DeploymentID:        "dep_target",
	}
}

// requireCode asserts err is a fault carrying the given precondition code.
func requireCode(t *testing.T, want codes.Code, err error) {
	t.Helper()
	require.Error(t, err)
	got, ok := fault.GetCode(err)
	require.True(t, ok, "expected a coded fault")
	require.Equal(t, want.URN(), got)
}

// ctrl surfaces fault.UserFacingMessage(err) on its own error surface, so verify
// the faults deploygate builds expose exactly the public message — not the base
// "precondition failed" text or the internal detail.
func TestFaultUserFacingMessage(t *testing.T) {
	in := promoteBase()
	in.Status = "building"
	require.Equal(t, "The deployment is not ready.", fault.UserFacingMessage(CheckPromoteTarget(in)))

	in = promoteBase()
	in.CurrentDeploymentID = in.DeploymentID
	require.Equal(t, "The deployment is already the current deployment.", fault.UserFacingMessage(CheckPromoteTarget(in)))

	stop := StopInput{Status: dbtype.DeploymentsStatusReady, DesiredState: dbtype.DeploymentsDesiredStateStopped}
	require.Equal(t, "The deployment is already stopping.", fault.UserFacingMessage(CheckStopTarget(stop)))

	start := StartInput{DesiredState: dbtype.DeploymentsDesiredStateStopped, SpendSuspended: true}
	require.Equal(t, "The workspace is suspended by its Compute spend cap. Raise the spend limit to resume.", fault.UserFacingMessage(CheckStartTarget(start)))

	// The base and internal strings must never leak into the user-facing message.
	require.NotContains(t, fault.UserFacingMessage(CheckStopTarget(StopInput{})), "precondition failed")
}

func TestCheckPromoteTarget(t *testing.T) {
	t.Run("eligible", func(t *testing.T) {
		require.NoError(t, CheckPromoteTarget(promoteBase()))
	})

	t.Run("reason order", func(t *testing.T) {
		in := promoteBase()
		in.Status = "building"
		requireCode(t, codes.App.Precondition.DeploymentNotReady, CheckPromoteTarget(in))

		in = promoteBase()
		in.DesiredState = "stopped"
		requireCode(t, codes.App.Precondition.DeploymentNotReady, CheckPromoteTarget(in))

		in = promoteBase()
		in.EnvironmentKind = dbtype.EnvironmentKindPreview
		requireCode(t, codes.App.Precondition.DeploymentNotProduction, CheckPromoteTarget(in))

		in = promoteBase()
		in.CurrentDeploymentID = ""
		requireCode(t, codes.App.Precondition.DeploymentNoCurrent, CheckPromoteTarget(in))
	})

	t.Run("already current is rejected", func(t *testing.T) {
		in := promoteBase()
		in.CurrentDeploymentID = in.DeploymentID
		requireCode(t, codes.App.Precondition.DeploymentIsCurrent, CheckPromoteTarget(in))
	})

	t.Run("promoting the current deployment while rolled back is allowed", func(t *testing.T) {
		in := promoteBase()
		in.CurrentDeploymentID = in.DeploymentID
		in.IsRolledBack = true
		require.NoError(t, CheckPromoteTarget(in), "confirming a rollback")
	})
}

func TestCheckRollbackTarget(t *testing.T) {
	t.Run("eligible", func(t *testing.T) {
		require.NoError(t, CheckRollbackTarget(rollbackBase()))
	})

	t.Run("rolling back to the current deployment is rejected", func(t *testing.T) {
		in := rollbackBase()
		in.CurrentDeploymentID = in.DeploymentID
		requireCode(t, codes.App.Precondition.DeploymentIsCurrent, CheckRollbackTarget(in))
	})

	t.Run("shares the core preconditions", func(t *testing.T) {
		in := rollbackBase()
		in.Status = "failed"
		requireCode(t, codes.App.Precondition.DeploymentNotReady, CheckRollbackTarget(in))
	})
}

func TestCheckStopTarget(t *testing.T) {
	running := StopInput{Status: dbtype.DeploymentsStatusReady, DesiredState: dbtype.DeploymentsDesiredStateRunning}

	require.NoError(t, CheckStopTarget(running))

	notReady := running
	notReady.Status = dbtype.DeploymentsStatusStopped
	requireCode(t, codes.App.Precondition.DeploymentNotRunning, CheckStopTarget(notReady))

	draining := running
	draining.DesiredState = dbtype.DeploymentsDesiredStateStopped
	requireCode(t, codes.App.Precondition.DeploymentIsStopping, CheckStopTarget(draining))

	prod := running
	prod.EnvironmentKind = dbtype.EnvironmentKindProduction
	requireCode(t, codes.App.Precondition.DeploymentIsProduction, CheckStopTarget(prod))
}

func TestCheckStartTarget(t *testing.T) {
	// A stopped deployment is keyed on desired_state, not status: it may still be
	// draining (status ready) while its intent is stopped.
	stopped := StartInput{DesiredState: dbtype.DeploymentsDesiredStateStopped}

	require.NoError(t, CheckStartTarget(stopped), "wakeable while draining toward stopped")

	notStopped := stopped
	notStopped.DesiredState = dbtype.DeploymentsDesiredStateRunning
	requireCode(t, codes.App.Precondition.DeploymentNotStopped, CheckStartTarget(notStopped))

	prod := stopped
	prod.EnvironmentKind = dbtype.EnvironmentKindProduction
	requireCode(t, codes.App.Precondition.DeploymentIsProduction, CheckStartTarget(prod))

	suspended := stopped
	suspended.SpendSuspended = true
	requireCode(t, codes.App.Precondition.PreconditionFailed, CheckStartTarget(suspended))
}
