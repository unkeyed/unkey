package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_create_identity"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestCreateIdentityDuplicate(t *testing.T) {
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
	})

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources.UserWorkspace.ID, "identity.*.create_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	externalTestID := "external_test_duplicate"
	t.Run("create identity twice", func(t *testing.T) {
		req := handler.Request{ExternalId: externalTestID}

		successRes := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, successRes.Status, "expected 200, received: %#v", successRes.Body)
		require.NotNil(t, successRes.Body)
		require.NotEmpty(t, successRes.Body.IdentityId)

		errorRes := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 409, errorRes.Status, "expected 200, received: %#v", errorRes)
		require.NotNil(t, errorRes.Body)
		require.NotEmpty(t, errorRes.Body.IdentityId)
	})
}
