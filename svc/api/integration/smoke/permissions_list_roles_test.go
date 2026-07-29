package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func TestListRoles_ReturnsCreatedRole(t *testing.T) {
	ctx, client := externalClient(t)
	createRole(t, ctx, client)
	response, err := client.Permissions.ListRoles(ctx, components.V2PermissionsListRolesRequestBody{Limit: ptr.P(int64(10))})
	require.NoError(t, err)
	require.NotNil(t, response.V2PermissionsListRolesResponseBody)
	require.NotEmpty(t, response.V2PermissionsListRolesResponseBody.Data)
}
