package handler

import (
	"fmt"
	"net/http"
	"testing"
	"time"

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

// bufferRequest queues one gateway request row. It fills the internal columns
// so a test can prove they stay unreachable even when they hold data.
func bufferRequest(t *testing.T, h *testutil.Harness, row schema.FrontlineRequest) schema.FrontlineRequest {
	t.Helper()

	if row.RequestID == "" {
		row.RequestID = uid.New(uid.RequestPrefix)
	}
	if row.Time == 0 {
		row.Time = time.Now().UnixMilli()
	}
	if row.ProjectID == "" {
		row.ProjectID = uid.New("proj")
	}
	if row.AppID == "" {
		row.AppID = uid.New("app")
	}
	if row.EnvironmentID == "" {
		row.EnvironmentID = uid.New("env")
	}
	row.FrontlineID = "frontline_internal"
	row.InstanceAddress = "10.1.2.3"
	row.Platform = "k8s"

	h.FrontlineRequests.Buffer(row)
	return row
}

func auth(rootKey string) http.Header {
	return http.Header{"Authorization": {fmt.Sprintf("Bearer %s", rootKey)}, "Content-Type": {"application/json"}}
}
