package deployment

import (
	"database/sql"
	"testing"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestAvailableActions(t *testing.T) {
	const (
		ready    = mysqltype.DeploymentsStatusReady
		building = mysqltype.DeploymentsStatusBuilding
		stopped  = mysqltype.DeploymentsStatusStopped
		running  = mysqltype.DeploymentsDesiredStateRunning
		drained  = mysqltype.DeploymentsDesiredStateStopped
	)
	const self = "d_self"
	const other = "d_other"

	cases := []struct {
		name         string
		status       mysqltype.DeploymentsStatus
		desiredState mysqltype.DeploymentsDesiredState
		envSlug      string
		envKind      mysqltype.EnvironmentKind
		current      string // app's current_deployment_id
		rolledBack   bool
		want         []openapi.DeploymentAction
	}{
		{"production canary ready, another is live", ready, running, "canary", mysqltype.EnvironmentKindProduction, other, false,
			[]openapi.DeploymentAction{openapi.DeploymentActionPromote, openapi.DeploymentActionRollback, openapi.DeploymentActionStop}},
		{"production canary ready, this is live", ready, running, "canary", mysqltype.EnvironmentKindProduction, self, false,
			[]openapi.DeploymentAction{openapi.DeploymentActionStop}},
		{"production canary ready, this is the rolled-back live one", ready, running, "canary", mysqltype.EnvironmentKindProduction, self, true,
			// promote forward is legal, rollback to the current is not
			[]openapi.DeploymentAction{openapi.DeploymentActionPromote, openapi.DeploymentActionStop}},
		{"production canary ready, app has no live deployment", ready, running, "canary", mysqltype.EnvironmentKindProduction, "", false,
			[]openapi.DeploymentAction{openapi.DeploymentActionStop}},
		{"production canary ready but draining", ready, drained, "canary", mysqltype.EnvironmentKindProduction, other, false,
			[]openapi.DeploymentAction{}},
		{"production canary still building", building, running, "canary", mysqltype.EnvironmentKindProduction, other, false,
			[]openapi.DeploymentAction{}},
		{"production stopped", stopped, drained, "production", mysqltype.EnvironmentKindProduction, other, false,
			[]openapi.DeploymentAction{openapi.DeploymentActionStart}},
		{"preview ready running", ready, running, "preview", mysqltype.EnvironmentKindPreview, "", false,
			[]openapi.DeploymentAction{openapi.DeploymentActionStop}},
		{"preview stopped", stopped, drained, "preview", mysqltype.EnvironmentKindPreview, "", false,
			[]openapi.DeploymentAction{openapi.DeploymentActionStart}},
		{"preview building", building, running, "preview", mysqltype.EnvironmentKindPreview, "", false,
			[]openapi.DeploymentAction{}},
		{"unknown environment offers nothing", ready, running, "", "", other, false,
			[]openapi.DeploymentAction{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			workspaceID := uid.New(uid.WorkspacePrefix)
			got := availableActions(Input{
				WorkspaceID: workspaceID,
				Permissions: []string{
					rbac.U(urn.New().Workspace(workspaceID).Project("*").App("*").Environment("*"), permissions.Write).Value,
					rbac.U(urn.New().Workspace(workspaceID).Project("*").App("*").Environment("*").Deployment("*"), permissions.Write).Value,
				},
				Deployment: db.ListDeploymentsRow{ID: self, Status: tc.status, DesiredState: tc.desiredState, ProjectID: uid.New(uid.ProjectPrefix), AppID: uid.New(uid.AppPrefix), EnvironmentID: uid.New(uid.EnvironmentPrefix)},
				State: db.ListDeploymentEnvAndAppStateRow{
					EnvironmentSlug:        tc.envSlug,
					EnvironmentKind:        tc.envKind,
					AppCurrentDeploymentID: sql.NullString{Valid: tc.current != "", String: tc.current},
					AppIsRolledBack:        tc.rolledBack,
				},
			})
			require.Equal(t, tc.want, got)
		})
	}
}

// TestAvailableActionsFollowPermissions pins that each action is offered only
// with the permission its handler checks: write on the environment for promote
// and rollback, write on the deployment for stop and start.
func TestAvailableActionsFollowPermissions(t *testing.T) {
	workspaceID := uid.New(uid.WorkspacePrefix)
	environment := urn.New().Workspace(workspaceID).Project("*").App("*").Environment("*")
	writeEnvironment := rbac.U(environment, permissions.Write).Value
	writeDeployment := rbac.U(environment.Deployment("*"), permissions.Write).Value
	readDeployment := rbac.U(environment.Deployment("*"), permissions.Read).Value

	production := db.ListDeploymentEnvAndAppStateRow{
		EnvironmentSlug:        "production",
		EnvironmentKind:        mysqltype.EnvironmentKindProduction,
		AppCurrentDeploymentID: sql.NullString{Valid: true, String: uid.New(uid.DeploymentPrefix)},
	}
	preview := db.ListDeploymentEnvAndAppStateRow{
		EnvironmentSlug: "preview",
		EnvironmentKind: mysqltype.EnvironmentKindPreview,
	}

	cases := []struct {
		name        string
		permissions []string
		state       db.ListDeploymentEnvAndAppStateRow
		want        []openapi.DeploymentAction
	}{
		{"environment write offers promote and rollback", []string{writeEnvironment}, production,
			[]openapi.DeploymentAction{openapi.DeploymentActionPromote, openapi.DeploymentActionRollback}},
		{"environment write does not offer stop", []string{writeEnvironment}, preview,
			[]openapi.DeploymentAction{}},
		{"deployment write offers stop", []string{writeDeployment}, preview,
			[]openapi.DeploymentAction{openapi.DeploymentActionStop}},
		{"deployment write offers only stop in production", []string{writeDeployment}, production,
			[]openapi.DeploymentAction{openapi.DeploymentActionStop}},
		{"read offers nothing", []string{readDeployment}, preview,
			[]openapi.DeploymentAction{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := availableActions(Input{
				WorkspaceID: workspaceID,
				Permissions: tc.permissions,
				Deployment: db.ListDeploymentsRow{
					ID:            uid.New(uid.DeploymentPrefix),
					Status:        mysqltype.DeploymentsStatusReady,
					DesiredState:  mysqltype.DeploymentsDesiredStateRunning,
					ProjectID:     uid.New(uid.ProjectPrefix),
					AppID:         uid.New(uid.AppPrefix),
					EnvironmentID: uid.New(uid.EnvironmentPrefix),
				},
				State: tc.state,
			})
			require.Equal(t, tc.want, got)
		})
	}
}
