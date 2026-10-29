package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_promote_deployment"
)

func newRoute(h *testutil.Harness, restate *restateingress.Client) *handler.Handler {
	return &handler.Handler{
		DB:      h.DB,
		Restate: restate,
	}
}

func environmentRead(setup testutil.DeploymentTestSetup) string {
	return fmt.Sprintf("unkey:v1:%s:projects/%s/apps/%s/environments/%s#read",
		setup.Workspace.ID, setup.Project.ID, setup.App.ID, setup.Environment.ID)
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + rootKey},
	}
}

// setCurrentDeployment marks a deployment as the app's live deployment,
// mimicking what ctrl persists after a promotion.
func setCurrentDeployment(t *testing.T, h *testutil.Harness, appID, deploymentID string) {
	t.Helper()
	setCurrentDeploymentState(t, h, appID, deploymentID, false)
}

// markRolledBack marks a deployment as the app's live deployment in a
// rolled-back state, mimicking what ctrl persists after a rollback.
func markRolledBack(t *testing.T, h *testutil.Harness, appID, deploymentID string) {
	t.Helper()
	setCurrentDeploymentState(t, h, appID, deploymentID, true)
}

func setCurrentDeploymentState(t *testing.T, h *testutil.Harness, appID, deploymentID string, rolledBack bool) {
	t.Helper()
	err := db.Query.UpdateAppDeployments(context.Background(), h.DB.RW(), db.UpdateAppDeploymentsParams{
		CurrentDeploymentID: sql.NullString{String: deploymentID, Valid: true},
		IsRolledBack:        rolledBack,
		UpdatedAt:           sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		AppID:               appID,
	})
	require.NoError(t, err)
}

// writeEnvironments is the permission to write every environment, which this
// handler checks.
func writeEnvironments(workspaceID string) string {
	return rbac.U(urn.New().Workspace(workspaceID).Project("*").App("*").Environment("*"), permissions.Write).Value
}
