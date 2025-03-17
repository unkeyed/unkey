package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_create_identity"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestCreateIdentityDuplicate(t *testing.T) {
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
	})

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources.UserWorkspace.ID, "identity.*.create_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("create identity twice", func(t *testing.T) {
		req := handler.Request{ExternalId: uid.New("test_external_id")}

		successRes := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, successRes.Status, "expected 200, received: %#v", successRes.Body)
		require.NotNil(t, successRes.Body)
		require.NotEmpty(t, successRes.Body.IdentityId, successRes.Body)

		errorRes := testutil.CallRoute[handler.Request, openapi.ConflictError](h, route, headers, req)
		require.Equal(t, 409, errorRes.Status, "expected 409, received: %#v", errorRes)
		require.NotNil(t, errorRes.Body)
		require.Equal(t, "https://unkey.com/docs/errors/conflict", errorRes.Body.Type)
	})
}
