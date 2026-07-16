package deployment

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestAvailableActions(t *testing.T) {
	const (
		ready    = db.DeploymentsStatusReady
		building = db.DeploymentsStatusBuilding
		stopped  = db.DeploymentsStatusStopped
		running  = db.DeploymentsDesiredStateRunning
		drained  = db.DeploymentsDesiredStateStopped
	)
	const self = "d_self"
	const other = "d_other"

	cases := []struct {
		name         string
		status       db.DeploymentsStatus
		desiredState db.DeploymentsDesiredState
		envSlug      string
		current      string // app's current_deployment_id
		rolledBack   bool
		want         []openapi.DeploymentAction
	}{
		{"prod ready, another is live", ready, running, "production", other, false,
			[]openapi.DeploymentAction{openapi.DeploymentActionPromote, openapi.DeploymentActionRollback}},
		{"prod ready, this is live", ready, running, "production", self, false,
			[]openapi.DeploymentAction{}},
		{"prod ready, this is the rolled-back live one", ready, running, "production", self, true,
			// promote forward is legal, rollback to the current is not
			[]openapi.DeploymentAction{openapi.DeploymentActionPromote}},
		{"prod ready, app has no live deployment", ready, running, "production", "", false,
			[]openapi.DeploymentAction{}},
		{"prod ready but draining", ready, drained, "production", other, false,
			[]openapi.DeploymentAction{}},
		{"prod still building", building, running, "production", other, false,
			[]openapi.DeploymentAction{}},
		{"preview ready running", ready, running, "preview", "", false,
			[]openapi.DeploymentAction{openapi.DeploymentActionStop}},
		{"preview stopped", stopped, drained, "preview", "", false,
			[]openapi.DeploymentAction{openapi.DeploymentActionStart}},
		{"preview building", building, running, "preview", "", false,
			[]openapi.DeploymentAction{}},
		{"unknown environment offers nothing", ready, running, "", other, false,
			[]openapi.DeploymentAction{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := availableActions(Input{
				Deployment:             db.Deployment{ID: self, Status: tc.status, DesiredState: tc.desiredState},
				EnvironmentSlug:        tc.envSlug,
				AppCurrentDeploymentID: tc.current,
				AppIsRolledBack:        tc.rolledBack,
			})
			require.Equal(t, tc.want, got)
		})
	}
}
