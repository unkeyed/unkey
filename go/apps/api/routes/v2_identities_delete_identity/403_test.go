package handler_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_delete_identity"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestWorkspacePermissions(t *testing.T) {
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
	})

	h.Register(route)

	t.Run("insufficient permissions", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID)
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{ExternalId: ptr.P(uid.New("test"))}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status, "got: %s", res.RawBody)
		require.NotNil(t, res.Body)
	})

	t.Run("delete identity from other workspace", func(t *testing.T) {
		identityId := uid.New(uid.IdentityPrefix)

		err := db.Query.InsertIdentity(t.Context(), h.DB.RW(), db.InsertIdentityParams{
			ID:          identityId,
			ExternalID:  "ext_" + identityId,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Environment: "default",
			CreatedAt:   time.Now().Unix(),
			Meta:        nil,
		})
		require.NoError(t, err)

		rootKey := h.CreateRootKey(h.Resources().DifferentWorkspace.ID, "identity.*.delete_identity")
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{IdentityId: ptr.P(identityId)}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "got: %s", res.RawBody)
		require.NotNil(t, res.Body)
	})
}
