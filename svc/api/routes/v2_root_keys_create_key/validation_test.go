package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_root_keys_create_key"
)

// TestCreateIdentifiesInvalidDelegatedPermission guarantees the response points
// to the rejected request value without revealing why the permission is invalid.
func TestCreateIdentifiesInvalidDelegatedPermission(t *testing.T) {
	h, route, p := newHarness(t)
	validPermission := "unkey:v1:" + p.AuthorizedWorkspaceID + ":rootKeys/*#read"
	for _, test := range []struct {
		name       string
		permission string
	}{
		{name: "unsupported action", permission: "unkey:v1:" + p.AuthorizedWorkspaceID + ":rootKeys/*#decrypt"},
		{name: "foreign workspace", permission: "unkey:v1:ws_foreign:rootKeys/*#read"},
	} {
		t.Run(test.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, http.Header{
				"Authorization": {"Bearer test"},
				"Content-Type":  {"application/json"},
			}, handler.Request{Permissions: []string{validPermission, test.permission}})

			require.Equal(t, http.StatusBadRequest, res.Status, res.RawBody)
			require.Equal(t, openapi.BadRequestErrorResponse{
				Meta: openapi.Meta{RequestId: res.Body.Meta.RequestId},
				Error: openapi.BadRequestErrorDetails{
					Title:  "Bad Request",
					Type:   codes.App.Validation.InvalidInput.DocsURL(),
					Detail: "A requested permission is not a supported URN permission in this workspace.",
					Status: http.StatusBadRequest,
					Errors: []openapi.ValidationError{{
						Location: "body.permissions[1]",
						Message:  "The permission is not a supported URN permission in this workspace.",
					}},
				},
			}, *res.Body)
		})
	}
}
