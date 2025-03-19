//nolint:exhaustruct
package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_create_identity"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestBadRequests(t *testing.T) {
	metaData := make(map[string]*interface{})
	entriesNeeded := (handler.MAX_META_LENGTH_MB * 1024 * 1024) / 15
	for i := 0; i < entriesNeeded+1000; i++ {
		var data interface{} = fmt.Sprintf("some_%d", i)
		metaData[fmt.Sprintf("key_%d", i)] = &data
	}

	rawMeta, _ := json.Marshal(metaData)

	testCases := []struct {
		name          string
		req           openapi.V2IdentitiesCreateIdentityRequestBody
		expectedError openapi.BadRequestError
	}{
		{
			name: "missing external id",
			req:  openapi.V2IdentitiesCreateIdentityRequestBody{},
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
			name: "empty external id",
			req: openapi.V2IdentitiesCreateIdentityRequestBody{
				ExternalId: "",
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
			name: "too short identifier",
			req: openapi.V2IdentitiesCreateIdentityRequestBody{
				ExternalId: "ab",
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
			name: "too much metadata",
			req: openapi.V2IdentitiesCreateIdentityRequestBody{
				ExternalId: "abc",
				Meta:       &metaData,
			},
			expectedError: openapi.BadRequestError{
				Title:     "Bad Request",
				Detail:    fmt.Sprintf("Metadata is too large, it must be less than %dMB, got: %.2f", handler.MAX_META_LENGTH_MB, float64(len(rawMeta))/1024/1024),
				Status:    http.StatusBadRequest,
				Type:      "https://unkey.com/docs/errors/bad_request",
				Errors:    []openapi.ValidationError{},
				RequestId: "test",
				Instance:  nil,
			},
		},
		{
			name: "Invalid ratelimit",
			req: openapi.V2IdentitiesCreateIdentityRequestBody{
				ExternalId: "abc",
				Ratelimits: &[]openapi.V2Ratelimit{
					{
						Duration: 1,
						Limit:    1,
					},
				},
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

	h := testutil.NewHarness(t)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.create_identity")
	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
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
