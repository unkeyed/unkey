package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
)

func TestRemoveRoles_PersistsRemoval(t *testing.T) {
	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	key := createKey(t, ctx, client, api.APIID)
	role := createRole(t, ctx, client)
	_, err := client.Keys.AddRoles(ctx, components.V2KeysAddRolesRequestBody{KeyID: key.KeyID, Roles: []string{role.Name}})
	require.NoError(t, err)
	response, err := client.Keys.RemoveRoles(ctx, components.V2KeysRemoveRolesRequestBody{KeyID: key.KeyID, Roles: []string{role.Name}})
	require.NoError(t, err)
	require.NotNil(t, response.V2KeysRemoveRolesResponseBody)
	get, err := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
	require.NoError(t, err)
	require.NotNil(t, get.V2KeysGetKeyResponseBody)
	require.NotContains(t, get.V2KeysGetKeyResponseBody.Data.Roles, role.Name)
}
