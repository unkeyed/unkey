package handler_test

import (
	"fmt"
	"net/http"

	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/pkg/db"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_authorize_deployment"
)

func newRoute(h *testutil.Harness, restate *restateingress.Client) *handler.Handler {
	route := &handler.Handler{
		DB:      h.DB,
		Restate: restate,
	}
	h.Register(route)
	return route
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + rootKey},
	}
}

func seedDeployment(h *testutil.Harness, setup testutil.DeploymentTestSetup, status mysqltype.DeploymentsStatus) db.Deployment {
	return h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:                     uid.New(uid.DeploymentPrefix),
		WorkspaceID:            setup.Workspace.ID,
		ProjectID:              setup.Project.ID,
		AppID:                  setup.App.ID,
		EnvironmentID:          setup.Environment.ID,
		Status:                 status,
		ForkRepositoryFullName: "KEBAP/unkey",
	})
}

func writeAnyDeployment(workspaceID string) string {
	return "unkey:v1:" + workspaceID + ":projects/*/apps/*/environments/*/deployments/*#write"
}

func deploymentURN(setup testutil.DeploymentTestSetup, deploymentID, action string) string {
	return fmt.Sprintf(
		"unkey:v1:%s:projects/%s/apps/%s/environments/%s/deployments/%s#%s",
		setup.Workspace.ID, setup.Project.ID, setup.App.ID, setup.Environment.ID, deploymentID, action,
	)
}
