package smoke_test

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
)

func TestSetPermissions_PersistsAssignment(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	key := createKey(t, ctx, client, api.APIID)
	permission := createPermission(t, ctx, client)
	response, err := client.Keys.SetPermissions(ctx, components.V2KeysSetPermissionsRequestBody{KeyID: key.KeyID, Permissions: []string{permission.Slug}})
	require.NoError(t, err)
	require.NotNil(t, response.V2KeysSetPermissionsResponseBody)
	require.Eventually(t, func() bool {
		get, getErr := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
		return getErr == nil && get.V2KeysGetKeyResponseBody != nil && slices.Contains(get.V2KeysGetKeyResponseBody.Data.Permissions, permission.Slug)
	}, 30*time.Second, time.Second, "permission %q was not assigned to key", permission.Slug)
}
