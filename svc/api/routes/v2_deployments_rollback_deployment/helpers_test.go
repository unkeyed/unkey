package handler_test

import (
	"context"
	"database/sql"
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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_rollback_deployment"
)

func newRoute(h *testutil.Harness, restate *restateingress.Client) *handler.Handler {
	return &handler.Handler{
		DB:      h.DB,
		Restate: restate,
	}
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
	err := db.Query.UpdateAppDeployments(context.Background(), h.DB.RW(), db.UpdateAppDeploymentsParams{
		CurrentDeploymentID: sql.NullString{String: deploymentID, Valid: true},
		IsRolledBack:        false,
		UpdatedAt:           sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		AppID:               appID,
	})
	require.NoError(t, err)
}

// writeDeploymentEnv grants write on every environment and every deployment in
// it, which together are the grants the lifecycle handlers check.
func writeDeploymentEnv(workspaceID string) []string {
	return []string{
		rbac.U(urn.New().Workspace(workspaceID).Project("*").App("*").Environment("*"), permissions.Write).Value,
		rbac.U(urn.New().Workspace(workspaceID).Project("*").App("*").Environment("*").Deployment("*"), permissions.Write).Value,
	}
}

// readDeploymentEnv grants read at every level and no write, so every lifecycle
// handler refuses the caller.
func readDeploymentEnv(workspaceID string) string {
	return rbac.U(urn.New().Workspace(workspaceID).Project("*").App("*").Environment("*"), permissions.Read).Value
}
