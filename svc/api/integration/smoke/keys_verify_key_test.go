package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestVerifyKey_ReturnsValidResult(t *testing.T) {
	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	key := createKey(t, ctx, client, api.APIID)
	response, err := client.Keys.VerifyKey(ctx, components.V2KeysVerifyKeyRequestBody{Key: key.Key})
	require.NoError(t, err)
	require.NotNil(t, response.V2KeysVerifyKeyResponseBody)
	require.True(t, response.V2KeysVerifyKeyResponseBody.Data.Valid)
	require.NotNil(t, response.V2KeysVerifyKeyResponseBody.Data.KeyID)
	require.Equal(t, key.KeyID, *response.V2KeysVerifyKeyResponseBody.Data.KeyID)
}

func TestVerifyKey_EnforcesPermissions(t *testing.T) {
	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	key := createKey(t, ctx, client, api.APIID)
	permission := createPermission(t, ctx, client)
	_, err := client.Keys.AddPermissions(ctx, components.V2KeysAddPermissionsRequestBody{KeyID: key.KeyID, Permissions: []string{permission.Slug}})
	require.NoError(t, err)
	response, err := client.Keys.VerifyKey(ctx, components.V2KeysVerifyKeyRequestBody{Key: key.Key, Permissions: &permission.Slug})
	require.NoError(t, err)
	require.True(t, response.V2KeysVerifyKeyResponseBody.Data.Valid)
	require.Contains(t, response.V2KeysVerifyKeyResponseBody.Data.Permissions, permission.Slug)
	missing := uid.DNS1035()
	response, err = client.Keys.VerifyKey(ctx, components.V2KeysVerifyKeyRequestBody{Key: key.Key, Permissions: &missing})
	require.NoError(t, err)
	require.False(t, response.V2KeysVerifyKeyResponseBody.Data.Valid)
}

func TestVerifyKey_ReturnsMetadataAndIdentity(t *testing.T) {
	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	identity := createIdentity(t, ctx, client)
	meta := map[string]any{"smokeTest": uid.DNS1035()}
	created, err := client.Keys.CreateKey(ctx, components.V2KeysCreateKeyRequestBody{APIID: api.APIID, ExternalID: &identity.ExternalID, Meta: meta})
	require.NoError(t, err)
	key := created.V2KeysCreateKeyResponseBody.Data
	waitForPropagation()
	t.Cleanup(func() {
		_, err := client.Keys.DeleteKey(ctx, components.V2KeysDeleteKeyRequestBody{KeyID: key.KeyID, Permanent: ptr.P(true)})
		require.NoError(t, err)
	})
	response, err := client.Keys.VerifyKey(ctx, components.V2KeysVerifyKeyRequestBody{Key: key.Key})
	require.NoError(t, err)
	require.True(t, response.V2KeysVerifyKeyResponseBody.Data.Valid)
	require.Equal(t, meta, response.V2KeysVerifyKeyResponseBody.Data.Meta)
	require.NotNil(t, response.V2KeysVerifyKeyResponseBody.Data.Identity)
	require.Equal(t, identity.ExternalID, response.V2KeysVerifyKeyResponseBody.Data.Identity.ExternalID)
}

func TestVerifyKey_ReturnsAutoAppliedRatelimit(t *testing.T) {
	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	limit := components.RatelimitRequest{Name: uid.DNS1035(), Limit: 10, Duration: 60_000, AutoApply: ptr.P(true)}
	created, err := client.Keys.CreateKey(ctx, components.V2KeysCreateKeyRequestBody{APIID: api.APIID, Ratelimits: []components.RatelimitRequest{limit}})
	require.NoError(t, err)
	key := created.V2KeysCreateKeyResponseBody.Data
	waitForPropagation()
	t.Cleanup(func() {
		_, err := client.Keys.DeleteKey(ctx, components.V2KeysDeleteKeyRequestBody{KeyID: key.KeyID, Permanent: ptr.P(true)})
		require.NoError(t, err)
	})
	response, err := client.Keys.VerifyKey(ctx, components.V2KeysVerifyKeyRequestBody{Key: key.Key})
	require.NoError(t, err)
	require.True(t, response.V2KeysVerifyKeyResponseBody.Data.Valid)
	require.Len(t, response.V2KeysVerifyKeyResponseBody.Data.Ratelimits, 1)
	require.Equal(t, limit.Name, response.V2KeysVerifyKeyResponseBody.Data.Ratelimits[0].Name)
	require.Equal(t, limit.Limit, response.V2KeysVerifyKeyResponseBody.Data.Ratelimits[0].Limit)
}
