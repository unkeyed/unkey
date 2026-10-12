package handler_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// TestGetFlagsReportsDatabaseFailure guarantees that a failed flag load returns
// an error instead of code defaults, so callers never mistake an outage for
// flags that are off.
func TestGetFlagsReportsDatabaseFailure(t *testing.T) {
	h := testutil.NewHarness(t)
	closedDatabase, err := db.New(db.Config{PrimaryDSN: containers.MySQL(t).DSN})
	require.NoError(t, err)
	require.NoError(t, closedDatabase.Close())
	route := newRoute(t, h, closedDatabase)
	h.Register(route)

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:flags#read", workspace.ID))

	req := httptest.NewRequest(route.Method(), route.Path(), nil)
	req.Header = bearer(rootKey)
	res := testutil.CallRaw[openapi.InternalServerErrorResponse](h, req)
	require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/application/service_unavailable", res.Body.Error.Type)
}
