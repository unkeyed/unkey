package handler_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_workspace_get_usage"
)

// failingClickHouse fails one of the parallel reads
type failingClickHouse struct {
	clickhouse.ClickHouse
}

func (failingClickHouse) GetBillableRatelimits(context.Context, string, int, int) (int64, error) {
	return 0, errors.New("KEBAP")
}

// panickingClickHouse panics in one of the parallel reads
type panickingClickHouse struct {
	clickhouse.ClickHouse
}

func (panickingClickHouse) GetActiveKeysByApp(context.Context, string, int, int) ([]clickhouse.ActiveKeysByApp, error) {
	panic("KEBAP")
}

// A panic in a read goroutine must fail only this request. Without recovery it
// stops the whole test binary
func TestGetUsageReadPanicReturns500(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:         h.DB,
		ClickHouse: panickingClickHouse{ClickHouse: clickhouse.NewNoop()},
		Clock:      h.Clock,
	}
	h.Register(route)

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:usage#read", workspace.ID))

	res := testutil.CallRoute[handler.Request, openapi.InternalServerErrorResponse](h, route, headers(rootKey), handler.Request{})
	require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)
}

func TestGetUsageReadErrorReturns500(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:         h.DB,
		ClickHouse: failingClickHouse{ClickHouse: clickhouse.NewNoop()},
		Clock:      h.Clock,
	}
	h.Register(route)

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:usage#read", workspace.ID))

	res := testutil.CallRoute[handler.Request, openapi.InternalServerErrorResponse](h, route, headers(rootKey), handler.Request{})
	require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)
	require.Equal(t, "Failed to read the workspace's usage.", res.Body.Error.Detail)
}
