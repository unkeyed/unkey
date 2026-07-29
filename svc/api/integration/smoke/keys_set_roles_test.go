package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
)

func TestSetRoles_PersistsAssignment(t *testing.T) {
	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	key := createKey(t, ctx, client, api.APIID)
	role := createRole(t, ctx, client)
	response, err := client.Keys.SetRoles(ctx, components.V2KeysSetRolesRequestBody{KeyID: key.KeyID, Roles: []string{role.Name}})
	require.NoError(t, err)
	require.NotNil(t, response.V2KeysSetRolesResponseBody)
	get, err := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
	require.NoError(t, err)
	require.NotNil(t, get.V2KeysGetKeyResponseBody)
	require.Contains(t, get.V2KeysGetKeyResponseBody.Data.Roles, role.Name)
}
