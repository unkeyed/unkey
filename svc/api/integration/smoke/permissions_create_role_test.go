package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestCreateRole_ReturnsRoleID(t *testing.T) {
	ctx, client := externalClient(t)
	response, err := client.Permissions.CreateRole(ctx, components.V2PermissionsCreateRoleRequestBody{Name: uid.DNS1035()})
	require.NoError(t, err)
	require.NotNil(t, response.V2PermissionsCreateRoleResponseBody)
	require.NotEmpty(t, response.V2PermissionsCreateRoleResponseBody.Data.RoleID)
	waitForPropagation()
	t.Cleanup(func() {
		_, err := client.Permissions.DeleteRole(ctx, components.V2PermissionsDeleteRoleRequestBody{Role: response.V2PermissionsCreateRoleResponseBody.Data.RoleID})
		require.NoError(t, err)
	})
}
