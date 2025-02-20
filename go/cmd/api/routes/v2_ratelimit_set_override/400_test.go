//nolint:exhaustruct
package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
	"github.com/unkeyed/unkey/go/api"
	handler "github.com/unkeyed/unkey/go/cmd/api/routes/v2_ratelimit_set_override"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestBadRequests(t *testing.T) {

	testCases := []struct {
		name          string
		req           api.V2RatelimitSetOverrideRequestBody
		expectedError api.BadRequestError
	}{
		{
			name: "missing all required fields",
			req:  api.V2RatelimitSetOverrideRequestBody{},
			expectedError: api.BadRequestError{
				Title:     "Bad Request",
				Detail:    "One or more fields failed validation",
				Status:    http.StatusBadRequest,
				Type:      "https://unkey.com/docs/errors/bad_request",
				Errors:    []api.ValidationError{},
				RequestId: "test",
				Instance:  nil,
			},
		},
		{
			name: "missing identifier",
			req: api.V2RatelimitSetOverrideRequestBody{
				NamespaceId: util.Pointer("not_empty"),
				Duration:    1000,
				Limit:       100,
			},
			expectedError: api.BadRequestError{
				Title:     "Bad Request",
				Detail:    "One or more fields failed validation",
				Status:    http.StatusBadRequest,
				Type:      "https://unkey.com/docs/errors/bad_request",
				Errors:    []api.ValidationError{},
				RequestId: "test",
				Instance:  nil,
			},
		},
		{
			name: "missing duration",
			req: api.V2RatelimitSetOverrideRequestBody{
				NamespaceId: util.Pointer("not_empty"),
				Identifier:  "user_123",
				Limit:       100,
			},
			expectedError: api.BadRequestError{
				Title:     "Bad Request",
				Detail:    "One or more fields failed validation",
				Status:    http.StatusBadRequest,
				Type:      "https://unkey.com/docs/errors/bad_request",
				Errors:    []api.ValidationError{},
				RequestId: "test",
				Instance:  nil,
			},
		},
		{
			name: "missing limit",
			req: api.V2RatelimitSetOverrideRequestBody{
				NamespaceId:   util.Pointer("not_empty"),
				NamespaceName: nil,
				Identifier:    "user_123",
				Duration:      1000,
			},
			expectedError: api.BadRequestError{
				Title:     "Bad Request",
				Detail:    "One or more fields failed validation",
				Status:    http.StatusBadRequest,
				Type:      "https://unkey.com/docs/errors/bad_request",
				Errors:    []api.ValidationError{},
				RequestId: "test",
				Instance:  nil,
			},
		},
		{
			name: "negative duration",
			req: api.V2RatelimitSetOverrideRequestBody{
				NamespaceId:   util.Pointer("not_empty"),
				NamespaceName: nil,
				Identifier:    "user_123",
				Duration:      -1000,
				Limit:         100,
			},
			expectedError: api.BadRequestError{
				Title:     "Bad Request",
				Detail:    "One or more fields failed validation",
				Status:    http.StatusBadRequest,
				Type:      "https://unkey.com/docs/errors/bad_request",
				Errors:    []api.ValidationError{},
				RequestId: "test",
				Instance:  nil,
			},
		},
		{
			name: "zero duration",
			req: api.V2RatelimitSetOverrideRequestBody{
				NamespaceId:   util.Pointer("not_empty"),
				NamespaceName: nil,
				Identifier:    "user_123",
				Duration:      0,
				Limit:         100,
			},
			expectedError: api.BadRequestError{
				Title:     "Bad Request",
				Detail:    "One or more fields failed validation",
				Status:    http.StatusBadRequest,
				Type:      "https://unkey.com/docs/errors/bad_request",
				Errors:    []api.ValidationError{},
				RequestId: "test",
				Instance:  nil,
			},
		},
		{
			name: "negative limit",
			req: api.V2RatelimitSetOverrideRequestBody{
				NamespaceId:   util.Pointer("not_empty"),
				NamespaceName: nil,
				Identifier:    "user_123",
				Duration:      1000,
				Limit:         -100,
			},
			expectedError: api.BadRequestError{
				Title:     "Bad Request",
				Detail:    "One or more fields failed validation",
				Status:    http.StatusBadRequest,
				Type:      "https://unkey.com/docs/errors/bad_request",
				Errors:    []api.ValidationError{},
				RequestId: "test",
				Instance:  nil,
			},
		},
		{
			name: "zero limit",
			req: api.V2RatelimitSetOverrideRequestBody{
				NamespaceId:   util.Pointer("not_empty"),
				NamespaceName: nil,
				Identifier:    "user_123",
				Duration:      1000,
				Limit:         0,
			},
			expectedError: api.BadRequestError{
				Title:     "Bad Request",
				Detail:    "One or more fields failed validation",
				Status:    http.StatusBadRequest,
				Type:      "https://unkey.com/docs/errors/bad_request",
				Errors:    []api.ValidationError{},
				RequestId: "test",
				Instance:  nil,
			},
		},
		{
			name: "empty identifier",
			req: api.V2RatelimitSetOverrideRequestBody{
				NamespaceId:   util.Pointer("not_empty"),
				NamespaceName: nil,
				Identifier:    "",
				Duration:      1000,
				Limit:         100,
			},
			expectedError: api.BadRequestError{
				Title:     "Bad Request",
				Detail:    "One or more fields failed validation",
				Status:    http.StatusBadRequest,
				Type:      "https://unkey.com/docs/errors/bad_request",
				Errors:    []api.ValidationError{},
				RequestId: "test",
				Instance:  nil,
			},
		},
		{
			name: "neither namespace ID nor name provided",
			req: api.V2RatelimitSetOverrideRequestBody{
				NamespaceId:   nil,
				NamespaceName: nil,
				Identifier:    "user_123",
				Duration:      1000,
				Limit:         100,
			},
			expectedError: api.BadRequestError{
				Title:     "Bad Request",
				Detail:    "You must provide either a namespace ID or name.",
				Status:    http.StatusBadRequest,
				Type:      "https://unkey.com/docs/errors/bad_request",
				Errors:    []api.ValidationError{},
				RequestId: "test",
				Instance:  nil,
			},
		},
	}

	for _, tc := range testCases {

		t.Run(tc.name, func(t *testing.T) {
			h := testutil.NewHarness(t)

			rootKey := h.CreateRootKey()
			route := handler.New(handler.Services{
				DB:     h.DB,
				Keys:   h.Keys,
				Logger: h.Logger,
			})

			h.Register(route)

			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, tc.req)
			require.Equal(t, 400, res.Status, "expected 400, received: %v", res.Body)
			require.NotNil(t, res.ErrorBody)
			require.Equal(t, tc.expectedError.Type, res.ErrorBody.Type)
			require.Equal(t, tc.expectedError.Detail, res.ErrorBody.Detail)
			require.Equal(t, tc.expectedError.Status, res.ErrorBody.Status)
			require.Equal(t, tc.expectedError.Title, res.ErrorBody.Title)
			require.NotEmpty(t, res.ErrorBody.RequestId)

		})
	}

}
