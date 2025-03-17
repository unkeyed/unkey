package v2RatelimitLimit_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_limit"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestBadRequests(t *testing.T) {
	testCases := []struct {
		name          string
		req           openapi.V2RatelimitLimitRequestBody
		expectedError openapi.BadRequestError
	}{
		//	{
		//		name: "missing namespace",
		//		req: openapi.V2RatelimitLimitRequestBody{
		//			Identifier: "user_123",
		//			Limit:      100,
		//			Duration:   60000,
		//		},
		//		expectedError: openapi.BadRequestError{
		//			Title:     "Bad Request",
		//			Detail:    "One or more fields failed validation",
		//			Status:    http.StatusBadRequest,
		//			Type:      "https://unkey.com/docs/errors/bad_request",
		//			Errors:    []openapi.ValidationError{},
		//			RequestId: "test",
		//			Instance:  nil,
		//		},
		//	},
		// {
		// 	name: "missing identifier",
		// 	req: openapi.V2RatelimitLimitRequestBody{
		// 		Namespace: "test_namespace",
		// 		Limit:     100,
		// 		Duration:  60000,
		// 	},
		// 	expectedError: openapi.BadRequestError{
		// 		Title:     "Bad Request",
		// 		Detail:    "One or more fields failed validation",
		// 		Status:    http.StatusBadRequest,
		// 		Type:      "https://unkey.com/docs/errors/bad_request",
		// 		Errors:    []openapi.ValidationError{},
		// 		RequestId: "test",
		// 		Instance:  nil,
		// 	},
		// },
		// {
		// 	name: "missing limit",
		// 	req: openapi.V2RatelimitLimitRequestBody{
		// 		Namespace:  "test_namespace",
		// 		Identifier: "user_123",
		// 		Duration:   60000,
		// 	},
		// 	expectedError: openapi.BadRequestError{
		// 		Title:     "Bad Request",
		// 		Detail:    "One or more fields failed validation",
		// 		Status:    http.StatusBadRequest,
		// 		Type:      "https://unkey.com/docs/errors/bad_request",
		// 		Errors:    []openapi.ValidationError{},
		// 		RequestId: "test",
		// 		Instance:  nil,
		// 	},
		// },
		// {
		// 	name: "missing duration",
		// 	req: openapi.V2RatelimitLimitRequestBody{
		// 		Namespace:  "test_namespace",
		// 		Identifier: "user_123",
		// 		Limit:      100,
		// 	},
		// 	expectedError: openapi.BadRequestError{
		// 		Title:     "Bad Request",
		// 		Detail:    "One or more fields failed validation",
		// 		Status:    http.StatusBadRequest,
		// 		Type:      "https://unkey.com/docs/errors/bad_request",
		// 		Errors:    []openapi.ValidationError{},
		// 		RequestId: "test",
		// 		Instance:  nil,
		// 	},
		// },
		// {
		// 	name: "negative limit",
		// 	req: openapi.V2RatelimitLimitRequestBody{
		// 		Namespace:  "test_namespace",
		// 		Identifier: "user_123",
		// 		Limit:      -1,
		// 		Duration:   60000,
		// 	},
		// 	expectedError: openapi.BadRequestError{
		// 		Title:     "Bad Request",
		// 		Detail:    "One or more fields failed validation",
		// 		Status:    http.StatusBadRequest,
		// 		Type:      "https://unkey.com/docs/errors/bad_request",
		// 		Errors:    []openapi.ValidationError{},
		// 		RequestId: "test",
		// 		Instance:  nil,
		// 	},
		// },
		// {
		// 	name: "negative duration",
		// 	req: openapi.V2RatelimitLimitRequestBody{
		// 		Namespace:  "test_namespace",
		// 		Identifier: "user_123",
		// 		Limit:      100,
		// 		Duration:   -1,
		// 	},
		// 	expectedError: openapi.BadRequestError{
		// 		Title:     "Bad Request",
		// 		Detail:    "One or more fields failed validation",
		// 		Status:    http.StatusBadRequest,
		// 		Type:      "https://unkey.com/docs/errors/bad_request",
		// 		Errors:    []openapi.ValidationError{},
		// 		RequestId: "test",
		// 		Instance:  nil,
		// 	},
		// },
		{
			name: "negative cost",
			req: openapi.V2RatelimitLimitRequestBody{
				Namespace:  "test_namespace",
				Identifier: "user_123",
				Limit:      100,
				Duration:   60000,
				Cost:       ptr.P[int64](-5),
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
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			h := testutil.NewHarness(t)

			route := handler.New(handler.Services{
				DB:          h.DB,
				Keys:        h.Keys,
				Logger:      h.Logger,
				Permissions: h.Permissions,
				Ratelimit:   h.Ratelimit,
			})

			h.Register(route)

			namespace := db.InsertRatelimitNamespaceParams{
				ID:          uid.New(uid.TestPrefix),
				WorkspaceID: h.Resources.UserWorkspace.ID,
				Name:        tc.req.Namespace,
				CreatedAt:   time.Now().UnixMilli(),
			}
			if namespace.Name != "" {

				err := db.Query.InsertRatelimitNamespace(context.Background(), h.DB.RW(), namespace)
				require.NoError(t, err)
			}
			rootKey := h.CreateRootKey(h.Resources.UserWorkspace.ID, fmt.Sprintf("ratelimit.%s.limit", namespace.ID))

			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			res := testutil.CallRoute[handler.Request, openapi.BadRequestError](h, route, headers, tc.req)
			require.Equal(t, 400, res.Status, "expected 400, received: %s", res.RawBody)
			require.NotNil(t, res.Body)
			require.Equal(t, tc.expectedError.Type, res.Body.Type)
			require.Equal(t, tc.expectedError.Status, res.Body.Status)
			require.Equal(t, tc.expectedError.Title, res.Body.Title)
			require.NotEmpty(t, res.Body.RequestId)
		})
	}
}
