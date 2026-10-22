package deployment

import (
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
)

// Visibility is the read query for one deployment. It names that deployment
// and its environment, so a reader of either can tell the deployment exists
// without a grant on every environment.
func Visibility(workspaceID string, dep db.FindDeploymentWithEnvironmentRow) rbac.PermissionQuery {
	environment := urn.New().Workspace(workspaceID).Project(dep.ProjectID).App(dep.AppID).Environment(dep.EnvironmentID)
	return rbac.Or(
		rbac.U(environment.Deployment(dep.ID), permissions.Read),
		rbac.U(environment, permissions.Read),
	)
}
