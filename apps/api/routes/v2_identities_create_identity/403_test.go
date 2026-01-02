package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/api/openapi"
	handler "github.com/unkeyed/unkey/apps/api/routes/v2_identities_create_identity"
	"github.com/unkeyed/unkey/pkg/testutil"
)

func TestWorkspacePermissions(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:    h.Logger,
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{ExternalId: "external_test_id"}
	res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
	require.Equal(t, http.StatusForbidden, res.Status, "got: %s", res.RawBody)
	require.NotNil(t, res.Body)
}
