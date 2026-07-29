package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
)

func TestRemovePermissions_PersistsRemoval(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	key := createKey(t, ctx, client, api.APIID)
	permission := createPermission(t, ctx, client)
	_, err := client.Keys.AddPermissions(ctx, components.V2KeysAddPermissionsRequestBody{KeyID: key.KeyID, Permissions: []string{permission.Slug}})
	require.NoError(t, err)
	waitForPropagation()
	response, err := client.Keys.RemovePermissions(ctx, components.V2KeysRemovePermissionsRequestBody{KeyID: key.KeyID, Permissions: []string{permission.Slug}})
	require.NoError(t, err)
	require.NotNil(t, response.V2KeysRemovePermissionsResponseBody)
	waitForPropagation()
	get, err := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
	require.NoError(t, err)
	require.NotNil(t, get.V2KeysGetKeyResponseBody)
	require.NotContains(t, get.V2KeysGetKeyResponseBody.Data.Permissions, permission.Slug)
}
