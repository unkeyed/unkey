package smoke_test

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
)

func TestSetRoles_PersistsAssignment(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	key := createKey(t, ctx, client, api.APIID)
	role := createRole(t, ctx, client)
	response, err := client.Keys.SetRoles(ctx, components.V2KeysSetRolesRequestBody{KeyID: key.KeyID, Roles: []string{role.Name}})
	require.NoError(t, err)
	require.NotNil(t, response.V2KeysSetRolesResponseBody)
	require.Eventually(t, func() bool {
		get, getErr := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
		return getErr == nil && get.V2KeysGetKeyResponseBody != nil && slices.Contains(get.V2KeysGetKeyResponseBody.Data.Roles, role.Name)
	}, 30*time.Second, time.Second, "role %q was not assigned to key", role.Name)
}
