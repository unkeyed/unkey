package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
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

func buildLogsRootKey(h *testutil.Harness, setup testutil.DeploymentTestSetup) string {
	everyDeployment := urn.New().Workspace(setup.Workspace.ID).Project("*").App("*").Environment("*").Deployment("*")
	return h.CreateRootKey(setup.Workspace.ID, rbac.U(everyDeployment.BuildLogs(), permissions.Read).Value)
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
	ctx := context.Background()
	batch, err := h.ClickHouse.Conn().PrepareBatch(ctx, clickhouse.InsertQuery[schema.BuildStepV1]())
	require.NoError(t, err)
	require.NoError(t, batch.AppendStruct(&schema.BuildStepV1{
		StartedAt:    startedAt,
		CompletedAt:  startedAt + 1,
		WorkspaceID:  target.workspaceID,
		ProjectID:    target.projectID,
		DeploymentID: target.deploymentID,
		StepID:       stepID,
		Name:         name,
		Cached:       false,
		Error:        "",
		HasLogs:      true,
	}))
	require.NoError(t, batch.Send())
}

// insertLogs writes count entries with seq from fromSeq and messages
// "KEBAP 0" to "KEBAP <count-1>"
func insertLogs(t *testing.T, h *testutil.Harness, target buildTarget, stepID string, time int64, fromSeq uint64, count int, stderr bool) {
	t.Helper()
	ctx := context.Background()
	batch, err := h.ClickHouse.Conn().PrepareBatch(ctx, clickhouse.InsertQuery[schema.BuildStepLogV1]())
	require.NoError(t, err)
	for i := range count {
		require.NoError(t, batch.AppendStruct(&schema.BuildStepLogV1{
			Time:         time,
			WorkspaceID:  target.workspaceID,
			ProjectID:    target.projectID,
			DeploymentID: target.deploymentID,
			StepID:       stepID,
			Message:      fmt.Sprintf("KEBAP %d", i),
			Seq:          fromSeq + uint64(i),
			Stderr:       stderr,
		}))
	}
	require.NoError(t, batch.Send())
}
