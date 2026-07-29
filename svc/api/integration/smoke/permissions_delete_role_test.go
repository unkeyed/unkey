package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestDeleteRole_DeletesRole(t *testing.T) {
	ctx, client := externalClient(t)
	created, err := client.Permissions.CreateRole(ctx, components.V2PermissionsCreateRoleRequestBody{Name: uid.DNS1035()})
	require.NoError(t, err)
	require.NotNil(t, created.V2PermissionsCreateRoleResponseBody)
	deleted, err := client.Permissions.DeleteRole(ctx, components.V2PermissionsDeleteRoleRequestBody{Role: created.V2PermissionsCreateRoleResponseBody.Data.RoleID})
	require.NoError(t, err)
	require.NotNil(t, deleted.V2PermissionsDeleteRoleResponseBody)
}
