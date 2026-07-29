package smoke_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
)

func TestRemoveRoles_PersistsRemoval(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	key := createKey(t, ctx, client, api.APIID)
	role := createRole(t, ctx, client)
	_, err := client.Keys.AddRoles(ctx, components.V2KeysAddRolesRequestBody{KeyID: key.KeyID, Roles: []string{role.Name}})
	require.NoError(t, err)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		get, err := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
		require.NoError(c, err)
		require.NotNil(c, get.V2KeysGetKeyResponseBody)
		require.Contains(c, get.V2KeysGetKeyResponseBody.Data.Roles, role.Name)
	}, 30*time.Second, time.Second)
	response, err := client.Keys.RemoveRoles(ctx, components.V2KeysRemoveRolesRequestBody{KeyID: key.KeyID, Roles: []string{role.Name}})
	require.NoError(t, err)
	require.NotNil(t, response.V2KeysRemoveRolesResponseBody)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		get, err := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
		require.NoError(c, err)
		require.NotNil(c, get.V2KeysGetKeyResponseBody)
		require.NotContains(c, get.V2KeysGetKeyResponseBody.Data.Roles, role.Name)
	}, 30*time.Second, time.Second)
}
