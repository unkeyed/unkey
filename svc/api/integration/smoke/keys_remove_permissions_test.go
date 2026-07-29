package smoke_test

import (
	"slices"
	"testing"
	"time"

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
	require.Eventually(t, func() bool {
		get, getErr := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
		return getErr == nil && get.V2KeysGetKeyResponseBody != nil && slices.Contains(get.V2KeysGetKeyResponseBody.Data.Permissions, permission.Slug)
	}, 30*time.Second, time.Second, "permission %q was not assigned before removal", permission.Slug)
	response, err := client.Keys.RemovePermissions(ctx, components.V2KeysRemovePermissionsRequestBody{KeyID: key.KeyID, Permissions: []string{permission.Slug}})
	require.NoError(t, err)
	require.NotNil(t, response.V2KeysRemovePermissionsResponseBody)
	require.Eventually(t, func() bool {
		get, getErr := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
		return getErr == nil && get.V2KeysGetKeyResponseBody != nil && !slices.Contains(get.V2KeysGetKeyResponseBody.Data.Permissions, permission.Slug)
	}, 30*time.Second, time.Second, "permission %q was not removed from key", permission.Slug)
}
