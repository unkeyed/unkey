package validation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestSchemaIsValid(t *testing.T) {
	_, err := New()
	require.NoError(t, err)
}

func TestRolePermissionRequestCardinality(t *testing.T) {
	validator, err := New()
	require.NoError(t, err)

	for _, count := range []int{0, 1000, 1001} {
		permissions := make([]string, count)
		for i := range permissions {
			permissions[i] = fmt.Sprintf("permission.%d", i)
		}
		role := "bounded-role"
		for _, test := range []struct {
			path string
			body any
		}{
			{
				path: "/v2/permissions.createRole",
				body: openapi.V2PermissionsCreateRoleRequestBody{
					Name:        "bounded-role",
					Permissions: &permissions,
				},
			},
			{
				path: "/v2/permissions.setRolePermissions",
				body: openapi.V2PermissionsSetRolePermissionsRequestBody{
					Role:        &role,
					Permissions: permissions,
				},
			},
		} {
			t.Run(fmt.Sprintf("%s/%d", test.path, count), func(t *testing.T) {
				body, err := json.Marshal(test.body)
				require.NoError(t, err)
				req := httptest.NewRequest(http.MethodPost, test.path, bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer test-root-key")
				response, valid := validator.Validate(context.Background(), req)
				if count <= 1000 {
					require.True(t, valid, "%+v", response)
				} else {
					require.False(t, valid)
					require.Equal(t, http.StatusBadRequest, response.Error.Status)
					require.Contains(t, response.Error.Errors[0].Location, "permissions")
				}
			})
		}
	}
}
