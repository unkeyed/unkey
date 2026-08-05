package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_configuration"
)

func TestCreateConfigurationBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	rootKey := h.CreateRootKey(workspaceID)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	keyspaceID := uid.New(uid.KeySpacePrefix)
	appID := uid.New(uid.AppPrefix)

	t.Run("missing slug", func(t *testing.T) {
		req := handler.Request{KeyspaceId: &keyspaceID}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
	})

	t.Run("invalid slug", func(t *testing.T) {
		req := handler.Request{Slug: "Not_Valid", KeyspaceId: &keyspaceID}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
	})

	t.Run("neither keyspace nor app", func(t *testing.T) {
		req := handler.Request{Slug: "portal-nomap"}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, received: %s", res.RawBody)
	})

	t.Run("both keyspace and app", func(t *testing.T) {
		req := handler.Request{Slug: "portal-bothmap", KeyspaceId: &keyspaceID, AppId: &appID}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, received: %s", res.RawBody)
	})
}
