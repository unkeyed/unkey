package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
)

func TestGetPermission_ReturnsCreatedPermission(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	permission := createPermission(t, ctx, client)
	response, err := client.Permissions.GetPermission(ctx, components.V2PermissionsGetPermissionRequestBody{Permission: permission.ID})
	require.NoError(t, err)
	require.NotNil(t, response.V2PermissionsGetPermissionResponseBody)
	require.Equal(t, permission.Slug, response.V2PermissionsGetPermissionResponseBody.Data.Slug)
}
