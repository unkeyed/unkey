package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

func newRoute(t *testing.T, analytics bool) (*testutil.Harness, *Handler, string) {
	t.Helper()

	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	workspace := h.CreateWorkspace()
	if analytics {
		h.SetupAnalytics(workspace.ID)
	}
	route := &Handler{AnalyticsConnectionManager: h.AnalyticsConnectionManager}
	h.Register(route)
	return h, route, workspace.ID
}

// runtimeLog describes one row to write. A test sets only the fields that it
// asserts on. insertLog gives a value to the other fields.
type runtimeLog struct {
	logID         string
	workspaceID   string
	projectID     string
	environmentID string
	appID         string
	deploymentID  string
	severity      string
	message       string
	attributes    string
	time          int64
}

// insertLog writes one row and returns it with all its values.
//
// Vector is the only producer of this table. Thus there is no Go writer and no
// batch buffer. This batch goes directly to ClickHouse, and the row is visible
// to the next query. A test therefore needs no wait loop.
//
// platform and k8s_pod_name get a value that is not empty. If they were empty, a
// probe in the tests for the column permission could get an empty result and look
// successful when ClickHouse did not refuse it.
func insertLog(t *testing.T, h *testutil.Harness, row runtimeLog) runtimeLog {
	t.Helper()

	if row.logID == "" {
		row.logID = uid.New("log")
	}
	if row.projectID == "" {
		row.projectID = uid.New("proj")
	}
	if row.environmentID == "" {
		row.environmentID = uid.New("env")
	}
	if row.appID == "" {
		row.appID = uid.New("app")
	}
	if row.deploymentID == "" {
		row.deploymentID = uid.New("dep")
	}
	if row.severity == "" {
		row.severity = "info"
	}
	if row.attributes == "" {
		row.attributes = "{}"
	}
	if row.time == 0 {
		row.time = time.Now().UnixMilli()
	}

	batch, err := h.ClickHouse.Conn().PrepareBatch(context.Background(), clickhouse.InsertQuery[schema.RuntimeLogV1]())
	require.NoError(t, err)
	require.NoError(t, batch.AppendStruct(&schema.RuntimeLogV1{
		Time:          row.time,
		LogID:         row.logID,
		Severity:      row.severity,
		Message:       row.message,
		WorkspaceID:   row.workspaceID,
		ProjectID:     row.projectID,
		EnvironmentID: row.environmentID,
		AppID:         row.appID,
		DeploymentID:  row.deploymentID,
		K8sPodName:    "pod-" + uid.New("rep"),
		Region:        "local",
		Platform:      "k8s",
		Attributes:    json.RawMessage(row.attributes),
	}))
	require.NoError(t, batch.Send())

	return row
}

func auth(rootKey string) http.Header {
	return http.Header{"Authorization": {fmt.Sprintf("Bearer %s", rootKey)}, "Content-Type": {"application/json"}}
}
