//nolint:exhaustruct
package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/api"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_create_identity"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestBadRequests(t *testing.T) {
	metaData := make(map[string]*interface{}, 0)

	for i := range 1_000_000 {
		var data interface{} = fmt.Sprintf("some_%d", i)
		metaData[fmt.Sprintf("key_%d", i)] = &data
	}

	rawMeta, _ := json.Marshal(metaData)

	testCases := []struct {
		name          string
		req           api.V2IdentitiesCreateIdentityRequestBody
		expectedError api.BadRequestError
	}{
		{
			name: "missing external id",
			req:  api.V2IdentitiesCreateIdentityRequestBody{},
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
			name: "empty external id",
			req: api.V2IdentitiesCreateIdentityRequestBody{
				ExternalId: "",
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
			name: "too short identifier",
			req: api.V2IdentitiesCreateIdentityRequestBody{
				ExternalId: "ab",
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
			name: "too much metadata",
			req: api.V2IdentitiesCreateIdentityRequestBody{
				ExternalId: "abc",
				Meta:       &metaData,
			},
			expectedError: api.BadRequestError{
				Title:     "Bad Request",
				Detail:    fmt.Sprintf("Metadata is too large, it must be less than %dMB, got: %.2f", handler.MAX_META_LENGTH_MB, float64(len(rawMeta))/1024/1024),
				Status:    http.StatusBadRequest,
				Type:      "https://unkey.com/docs/errors/bad_request",
				Errors:    []api.ValidationError{},
				RequestId: "test",
				Instance:  nil,
			},
		},
	}

	h := testutil.NewHarness(t)

	rootKey := h.CreateRootKey(h.Resources.UserWorkspace.ID, "identity.*.create_identity")
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

			res := testutil.CallRoute[handler.Request, api.BadRequestError](h, route, headers, tc.req)
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
