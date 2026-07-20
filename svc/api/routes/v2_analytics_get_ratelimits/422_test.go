package handler

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

func Test422_ClickHouseResourceLimit(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID, testutil.WithMaxQueryMemoryBytes(10_000))
	id := createNamespace(t, h, workspace.ID)
	for i := range 50_000 {
		h.RatelimitEvents.Buffer(schema.Ratelimit{RequestID: uid.New(uid.RequestPrefix), Time: time.Now().UnixMilli(), WorkspaceID: workspace.ID, NamespaceID: id, Identifier: uid.New("user"), Passed: i%2 == 0, Limit: 100})
	}
	rootKey := h.CreateRootKey(workspace.ID, "ratelimit.*.read_analytics")
	route := &Handler{DB: h.DB, AnalyticsConnectionManager: h.AnalyticsConnectionManager}
	h.Register(route)
	time.Sleep(10 * time.Second)
	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: fmt.Sprintf("SELECT identifier, passed, count(*) FROM ratelimits_v1 WHERE namespace_id = '%s' GROUP BY identifier, passed", id)})
	require.Equal(t, 422, res.Status)
}
