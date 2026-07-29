package smoke_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	unkey "github.com/unkeyed/sdks/api/go/v2"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
)

func externalClient(t *testing.T) (context.Context, *unkey.Unkey) {
	t.Helper()

	rootKey := os.Getenv("UNKEY_ROOT_KEY")
	if rootKey == "" {
		t.Skip("set UNKEY_ROOT_KEY to run the API smoke tests")
	}

	options := []unkey.SDKOption{unkey.WithSecurity(rootKey)}
	if baseURL := os.Getenv("UNKEY_API_BASE_URL"); baseURL != "" {
		options = append(options, unkey.WithServerURL(baseURL))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)

	return ctx, unkey.New(options...)
}

func createAPI(t *testing.T, ctx context.Context, client *unkey.Unkey) components.V2ApisCreateAPIResponseData {
	t.Helper()

	response, err := client.Apis.CreateAPI(ctx, components.V2ApisCreateAPIRequestBody{
		Name: uid.DNS1035(),
	})
	require.NoError(t, err)
	require.NotNil(t, response.V2ApisCreateAPIResponseBody)
	api := response.V2ApisCreateAPIResponseBody.Data

	t.Cleanup(func() {
		_, err := client.Apis.DeleteAPI(ctx, components.V2ApisDeleteAPIRequestBody{APIID: api.APIID})
		require.NoError(t, err)
	})

	return api
}

func createKey(t *testing.T, ctx context.Context, client *unkey.Unkey, apiID string) components.V2KeysCreateKeyResponseData {
	t.Helper()

	name := uid.DNS1035()
	response, err := client.Keys.CreateKey(ctx, components.V2KeysCreateKeyRequestBody{
		APIID:   apiID,
		Name:    &name,
		Enabled: ptr.P(true),
	})
	require.NoError(t, err)
	require.NotNil(t, response.V2KeysCreateKeyResponseBody)
	key := response.V2KeysCreateKeyResponseBody.Data

	t.Cleanup(func() {
		_, err := client.Keys.DeleteKey(ctx, components.V2KeysDeleteKeyRequestBody{
			KeyID:     key.KeyID,
			Permanent: ptr.P(true),
		})
		require.NoError(t, err)
	})

	return key
}

func createPermission(t *testing.T, ctx context.Context, client *unkey.Unkey) components.Permission {
	t.Helper()

	name := uid.DNS1035()
	response, err := client.Permissions.CreatePermission(ctx, components.V2PermissionsCreatePermissionRequestBody{
		Name: name,
		Slug: name,
	})
	require.NoError(t, err)
	require.NotNil(t, response.V2PermissionsCreatePermissionResponseBody)
	permissionID := response.V2PermissionsCreatePermissionResponseBody.Data.PermissionID
	getResponse, err := client.Permissions.GetPermission(ctx, components.V2PermissionsGetPermissionRequestBody{
		Permission: permissionID,
	})
	require.NoError(t, err)
	require.NotNil(t, getResponse.V2PermissionsGetPermissionResponseBody)

	t.Cleanup(func() {
		_, err := client.Permissions.DeletePermission(ctx, components.V2PermissionsDeletePermissionRequestBody{
			Permission: permissionID,
		})
		require.NoError(t, err)
	})

	return getResponse.V2PermissionsGetPermissionResponseBody.Data
}

func createRole(t *testing.T, ctx context.Context, client *unkey.Unkey) components.Role {
	t.Helper()

	name := uid.DNS1035()
	response, err := client.Permissions.CreateRole(ctx, components.V2PermissionsCreateRoleRequestBody{
		Name: name,
	})
	require.NoError(t, err)
	require.NotNil(t, response.V2PermissionsCreateRoleResponseBody)
	roleID := response.V2PermissionsCreateRoleResponseBody.Data.RoleID
	getResponse, err := client.Permissions.GetRole(ctx, components.V2PermissionsGetRoleRequestBody{
		Role: roleID,
	})
	require.NoError(t, err)
	require.NotNil(t, getResponse.V2PermissionsGetRoleResponseBody)

	t.Cleanup(func() {
		_, err := client.Permissions.DeleteRole(ctx, components.V2PermissionsDeleteRoleRequestBody{
			Role: roleID,
		})
		require.NoError(t, err)
	})

	return getResponse.V2PermissionsGetRoleResponseBody.Data
}

func createIdentity(t *testing.T, ctx context.Context, client *unkey.Unkey) components.Identity {
	t.Helper()

	response, err := client.Identities.CreateIdentity(ctx, components.V2IdentitiesCreateIdentityRequestBody{
		ExternalID: uid.DNS1035(),
		Meta:       map[string]any{"smokeTest": uid.DNS1035()},
	})
	require.NoError(t, err)
	require.NotNil(t, response.V2IdentitiesCreateIdentityResponseBody)
	identityID := response.V2IdentitiesCreateIdentityResponseBody.Data.IdentityID
	getResponse, err := client.Identities.GetIdentity(ctx, components.V2IdentitiesGetIdentityRequestBody{
		Identity: identityID,
	})
	require.NoError(t, err)
	require.NotNil(t, getResponse.V2IdentitiesGetIdentityResponseBody)

	t.Cleanup(func() {
		_, err := client.Identities.DeleteIdentity(ctx, components.V2IdentitiesDeleteIdentityRequestBody{
			Identity: identityID,
		})
		require.NoError(t, err)
	})

	return getResponse.V2IdentitiesGetIdentityResponseBody.Data
}

func keyIDs(keys []components.KeyResponseData) []string {
	ids := make([]string, 0, len(keys))
	for _, key := range keys {
		ids = append(ids, key.KeyID)
	}
	return ids
}
