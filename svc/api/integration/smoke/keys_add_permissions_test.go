package smoke_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
)

func TestAddPermissions_PersistsAssignment(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	key := createKey(t, ctx, client, api.APIID)
	permission := createPermission(t, ctx, client)
	response, err := client.Keys.AddPermissions(ctx, components.V2KeysAddPermissionsRequestBody{KeyID: key.KeyID, Permissions: []string{permission.Slug}})
	require.NoError(t, err)
	require.NotNil(t, response.V2KeysAddPermissionsResponseBody)
	require.Contains(t, response.V2KeysAddPermissionsResponseBody.Data, permission)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		get, err := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
		require.NoError(c, err)
		require.NotNil(c, get.V2KeysGetKeyResponseBody)
		require.Contains(c, get.V2KeysGetKeyResponseBody.Data.Permissions, permission.Slug)
	}, 30*time.Second, time.Second)
}
