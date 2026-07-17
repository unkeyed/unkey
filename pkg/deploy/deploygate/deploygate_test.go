package deploygate

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
)

// TestEnumLiterals pins the package's string literals to the db-generated enum
// values, so a schema rename can't silently weaken the gates. pkg/db and the
// ctrl db package are generated from the same schema, so pinning one covers both.
func TestEnumLiterals(t *testing.T) {
	require.Equal(t, statusReady, string(db.DeploymentsStatusReady))
	require.Equal(t, statusStopped, string(db.DeploymentsStatusStopped))
	require.Equal(t, desiredRunning, string(db.DeploymentsDesiredStateRunning))
	require.Equal(t, desiredStopped, string(db.DeploymentsDesiredStateStopped))
}

func base() Input {
	return Input{
		Status:               statusReady,
		DesiredState:         desiredRunning,
		EnvironmentSlug:      envProduction,
		HasCurrentDeployment: true,
		CurrentDeploymentID:  "dep_live",
		DeploymentID:         "dep_target",
		IsRolledBack:         false,
	}
}

func TestCheckPromoteTarget(t *testing.T) {
	t.Run("eligible", func(t *testing.T) {
		require.Equal(t, PromotionOK, CheckPromoteTarget(base()))
	})

	t.Run("reason order", func(t *testing.T) {
		in := base()
		in.Status = "building"
		require.Equal(t, PromotionNotReady, CheckPromoteTarget(in))

		in = base()
		in.DesiredState = "stopped"
		require.Equal(t, PromotionDraining, CheckPromoteTarget(in))

		in = base()
		in.EnvironmentSlug = "preview"
		require.Equal(t, PromotionNotProduction, CheckPromoteTarget(in))

		in = base()
		in.HasCurrentDeployment = false
		require.Equal(t, PromotionNoCurrentDeployment, CheckPromoteTarget(in))
	})

	t.Run("already live is rejected", func(t *testing.T) {
		in := base()
		in.CurrentDeploymentID = in.DeploymentID
		require.Equal(t, PromotionAlreadyCurrent, CheckPromoteTarget(in))
	})

	t.Run("promoting the current deployment while rolled back is allowed", func(t *testing.T) {
		in := base()
		in.CurrentDeploymentID = in.DeploymentID
		in.IsRolledBack = true
		require.Equal(t, PromotionOK, CheckPromoteTarget(in), "confirming a rollback")
	})
}

func TestCheckRollbackTarget(t *testing.T) {
	t.Run("eligible", func(t *testing.T) {
		require.Equal(t, PromotionOK, CheckRollbackTarget(base()))
	})

	t.Run("rolling back to the current deployment is always rejected", func(t *testing.T) {
		in := base()
		in.CurrentDeploymentID = in.DeploymentID
		require.Equal(t, PromotionAlreadyCurrent, CheckRollbackTarget(in))

		// Unlike promote, the rolled-back exception does not apply.
		in.IsRolledBack = true
		require.Equal(t, PromotionAlreadyCurrent, CheckRollbackTarget(in))
	})

	t.Run("shares the core preconditions", func(t *testing.T) {
		in := base()
		in.Status = "failed"
		require.Equal(t, PromotionNotReady, CheckRollbackTarget(in))
	})
}

func TestCheckStoppable(t *testing.T) {
	running := Input{Status: statusReady, DesiredState: desiredRunning, EnvironmentSlug: "preview"}

	require.Equal(t, StopOK, CheckStoppable(running))

	notReady := running
	notReady.Status = statusStopped
	require.Equal(t, StopNotRunning, CheckStoppable(notReady))

	draining := running
	draining.DesiredState = desiredStopped
	require.Equal(t, StopAlreadyStopping, CheckStoppable(draining))

	prod := running
	prod.EnvironmentSlug = envProduction
	require.Equal(t, StopIsProduction, CheckStoppable(prod))
}

func TestCheckStartable(t *testing.T) {
	// A stopped deployment is keyed on desired_state, not status: it may still be
	// draining (status ready) while its intent is stopped.
	stopped := Input{Status: statusReady, DesiredState: desiredStopped, EnvironmentSlug: "preview"}

	require.Equal(t, StartOK, CheckStartable(stopped), "wakeable while draining toward stopped")

	notStopped := stopped
	notStopped.DesiredState = desiredRunning
	require.Equal(t, StartNotStopped, CheckStartable(notStopped))

	prod := stopped
	prod.EnvironmentSlug = envProduction
	require.Equal(t, StartIsProduction, CheckStartable(prod))
}
