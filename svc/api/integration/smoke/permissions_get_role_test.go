package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
)

func TestGetRole_ReturnsCreatedRole(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	role := createRole(t, ctx, client)
	response, err := client.Permissions.GetRole(ctx, components.V2PermissionsGetRoleRequestBody{Role: role.ID})
	require.NoError(t, err)
	require.NotNil(t, response.V2PermissionsGetRoleResponseBody)
	require.Equal(t, role.Name, response.V2PermissionsGetRoleResponseBody.Data.Name)
}
