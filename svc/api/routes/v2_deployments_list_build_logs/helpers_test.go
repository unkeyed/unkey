package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_list_build_logs"
)

func newRoute(h *testutil.Harness) *handler.Handler {
	return &handler.Handler{
		DB:         h.DB,
		ClickHouse: h.ClickHouse,
		Clock:      h.Clock,
	}
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + rootKey},
	}
}

type buildTarget struct {
	workspaceID, projectID, deploymentID string
}

func createDeployment(h *testutil.Harness, setup testutil.DeploymentTestSetup) buildTarget {
	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})
	return buildTarget{workspaceID: setup.Workspace.ID, projectID: setup.Project.ID, deploymentID: dep.ID}
}

// The ctrl worker writes these rows through a batch buffer. The tests write
// them directly so they are visible to the next query without a wait loop
func insertStep(t *testing.T, h *testutil.Harness, target buildTarget, stepID, name string, startedAt int64) {
	t.Helper()
	require.NoError(t, h.ClickHouse.Exec(context.Background(),
		"INSERT INTO default.build_steps_v1 (step_id, started_at, completed_at, workspace_id, project_id, deployment_id, name, cached, error, has_logs) VALUES (?, ?, ?, ?, ?, ?, ?, false, '', true)",
		stepID, startedAt, startedAt+1, target.workspaceID, target.projectID, target.deploymentID, name,
	))
}

// insertLogs writes count entries with seq from fromSeq and messages
// "KEBAP 0" to "KEBAP <count-1>"
func insertLogs(t *testing.T, h *testutil.Harness, target buildTarget, stepID string, time int64, fromSeq uint64, count int, stderr bool) {
	t.Helper()
	require.NoError(t, h.ClickHouse.Exec(context.Background(),
		"INSERT INTO default.build_step_logs_v1 (time, workspace_id, project_id, deployment_id, step_id, message, seq, stderr) SELECT toInt64(?), ?, ?, ?, ?, concat('KEBAP ', toString(number)), toUInt64(?) + number, ? FROM numbers(?)",
		time, target.workspaceID, target.projectID, target.deploymentID, stepID, fromSeq, stderr, count,
	))
}
