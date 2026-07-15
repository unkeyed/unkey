package handler_test

import (
	"context"
	"database/sql"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_promote_deployment"
)

func newRoute(h *testutil.Harness, mock *testutil.MockDeploymentClient) *handler.Handler {
	return &handler.Handler{
		DB:         h.DB,
		CtrlClient: mock,
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
