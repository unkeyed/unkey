//nolint:exhaustruct
package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_get_override"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestBadRequests(t *testing.T) {
	testCases := []struct {
		name          string
		req           openapi.V2RatelimitGetOverrideRequestBody
		expectedError openapi.BadRequestError
	}{
		{
			name: "missing all required fields",
			req:  openapi.V2RatelimitGetOverrideRequestBody{},
			expectedError: openapi.BadRequestError{
				Title:     "Bad Request",
				Detail:    "One or more fields failed validation",
				Status:    http.StatusBadRequest,
				Type:      "https://unkey.com/docs/errors/bad_request",
				Errors:    []openapi.ValidationError{},
				RequestId: "test",
				Instance:  nil,
			},
		},
		{
			name: "missing identifier",
			req: openapi.V2RatelimitGetOverrideRequestBody{
				NamespaceId: util.Pointer("not_empty"),
			},
			expectedError: openapi.BadRequestError{
				Title:     "Bad Request",
				Detail:    "One or more fields failed validation",
				Status:    http.StatusBadRequest,
				Type:      "https://unkey.com/docs/errors/bad_request",
				Errors:    []openapi.ValidationError{},
				RequestId: "test",
				Instance:  nil,
			},
		},
		{
			name: "empty identifier",
			req: openapi.V2RatelimitGetOverrideRequestBody{
				NamespaceId:   util.Pointer("not_empty"),
				NamespaceName: nil,
				Identifier:    "",
			},
			expectedError: openapi.BadRequestError{
				Title:     "Bad Request",
				Detail:    "One or more fields failed validation",
				Status:    http.StatusBadRequest,
				Type:      "https://unkey.com/docs/errors/bad_request",
				Errors:    []openapi.ValidationError{},
				RequestId: "test",
				Instance:  nil,
			},
		},
		{
			name: "neither namespace ID nor name provided",
			req: openapi.V2RatelimitGetOverrideRequestBody{
				NamespaceId:   nil,
				NamespaceName: nil,
				Identifier:    "user_123",
			},
			expectedError: openapi.BadRequestError{
				Title:     "Bad Request",
				Detail:    "You must provide either a namespace ID or name.",
				Status:    http.StatusBadRequest,
				Type:      "https://unkey.com/docs/errors/bad_request",
				Errors:    []openapi.ValidationError{},
				RequestId: "test",
				Instance:  nil,
			},
		},
	}

	h := testutil.NewHarness(t)

	rootKey := h.CreateRootKey(h.Resources.UserWorkspace.ID)
	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
	})

	h.Register(route)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			res := testutil.CallRoute[handler.Request, openapi.BadRequestError](h, route, headers, tc.req)
			require.Equal(t, 400, res.Status, "expected 400, received: %v", res.Body)
			require.NotNil(t, res.Body)
			require.Equal(t, tc.expectedError.Type, res.Body.Type)
			require.Equal(t, tc.expectedError.Detail, res.Body.Detail)
			require.Equal(t, tc.expectedError.Status, res.Body.Status)
			require.Equal(t, tc.expectedError.Title, res.Body.Title)
			require.NotEmpty(t, res.Body.RequestId)
		})
	}
}
