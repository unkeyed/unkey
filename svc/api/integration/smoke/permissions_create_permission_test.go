package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestCreatePermission_ReturnsPermissionID(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	name := uid.DNS1035()
	response, err := client.Permissions.CreatePermission(ctx, components.V2PermissionsCreatePermissionRequestBody{Name: name, Slug: name})
	require.NoError(t, err)
	require.NotNil(t, response.V2PermissionsCreatePermissionResponseBody)
	require.NotEmpty(t, response.V2PermissionsCreatePermissionResponseBody.Data.PermissionID)
	waitForPropagation()
	t.Cleanup(func() {
		_, err := client.Permissions.DeletePermission(ctx, components.V2PermissionsDeletePermissionRequestBody{Permission: response.V2PermissionsCreatePermissionResponseBody.Data.PermissionID})
		require.NoError(t, err)
	})
}
