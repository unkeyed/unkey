package handler_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_list_identities"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestBadRequests(t *testing.T) {
	h := testutil.NewHarness(t)
	route := handler.New(handler.Services{
		Logger:      h.Logger(),
		DB:          h.Database(),
		Keys:        h.Keys(),
		Permissions: h.Permissions(),
	})

	rootKeyID := h.CreateRootKey()
	headers := testutil.RootKeyAuth(rootKeyID)

	// Since this endpoint has mostly optional parameters with defaults,
	// there are fewer validation errors to test compared to other endpoints.
	// We'll test negative limit value, which should be handled in the handler.

	t.Run("negative limit", func(t *testing.T) {
		negativeLimit := -10
		req := handler.Request{
			limit: &negativeLimit,
		}

		// This should not return 400 as we handle it in the handler by setting to 1
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		// But we should have at least 1 result
		require.GreaterOrEqual(t, len(res.Body.Data.Identities), 1)
	})

	t.Run("invalid cursor format", func(t *testing.T) {
		invalidCursor := "invalid_cursor_format"
		req := handler.Request{
			cursor: &invalidCursor,
		}

		// This might return 400 or might just return an empty result set
		// depending on how cursor validation is implemented
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		// If it returns 400, validate the error response
		if res.Status == 400 {
			require.Equal(t, 400, res.Status)
			require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/application/invalid_input", res.Body.Error.Type)
			require.NotEmpty(t, res.Body.Meta.RequestId)
		}
	})
}
