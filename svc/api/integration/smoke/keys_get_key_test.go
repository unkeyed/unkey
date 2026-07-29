package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestGetKey_ReturnsBasicFields(t *testing.T) {
	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	key := createKey(t, ctx, client, api.APIID)
	response, err := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
	require.NoError(t, err)
	require.NotNil(t, response.V2KeysGetKeyResponseBody)
	require.Equal(t, key.KeyID, response.V2KeysGetKeyResponseBody.Data.KeyID)
	require.True(t, response.V2KeysGetKeyResponseBody.Data.Enabled)
	require.NotEmpty(t, response.V2KeysGetKeyResponseBody.Data.Start)
}

func TestGetKey_ReturnsPersistedMetadata(t *testing.T) {
	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	meta := map[string]any{"smokeTest": uid.DNS1035()}
	created, err := client.Keys.CreateKey(ctx, components.V2KeysCreateKeyRequestBody{APIID: api.APIID, Meta: meta})
	require.NoError(t, err)
	require.NotNil(t, created.V2KeysCreateKeyResponseBody)
	key := created.V2KeysCreateKeyResponseBody.Data
	waitForPropagation()
	t.Cleanup(func() {
		_, err := client.Keys.DeleteKey(ctx, components.V2KeysDeleteKeyRequestBody{KeyID: key.KeyID, Permanent: ptr.P(true)})
		require.NoError(t, err)
	})
	response, err := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
	require.NoError(t, err)
	require.NotNil(t, response.V2KeysGetKeyResponseBody)
	require.Equal(t, meta, response.V2KeysGetKeyResponseBody.Data.Meta)
}

func TestGetKey_ReturnsPersistedIdentity(t *testing.T) {
	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	identity := createIdentity(t, ctx, client)
	created, err := client.Keys.CreateKey(ctx, components.V2KeysCreateKeyRequestBody{APIID: api.APIID, ExternalID: &identity.ExternalID})
	require.NoError(t, err)
	require.NotNil(t, created.V2KeysCreateKeyResponseBody)
	key := created.V2KeysCreateKeyResponseBody.Data
	waitForPropagation()
	t.Cleanup(func() {
		_, err := client.Keys.DeleteKey(ctx, components.V2KeysDeleteKeyRequestBody{KeyID: key.KeyID, Permanent: ptr.P(true)})
		require.NoError(t, err)
	})
	response, err := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
	require.NoError(t, err)
	require.NotNil(t, response.V2KeysGetKeyResponseBody)
	require.NotNil(t, response.V2KeysGetKeyResponseBody.Data.Identity)
	require.Equal(t, identity.ExternalID, response.V2KeysGetKeyResponseBody.Data.Identity.ExternalID)
}

func TestGetKey_ReturnsPersistedRatelimit(t *testing.T) {
	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	limit := components.RatelimitRequest{Name: uid.DNS1035(), Limit: 10, Duration: 60_000, AutoApply: ptr.P(true)}
	created, err := client.Keys.CreateKey(ctx, components.V2KeysCreateKeyRequestBody{APIID: api.APIID, Ratelimits: []components.RatelimitRequest{limit}})
	require.NoError(t, err)
	require.NotNil(t, created.V2KeysCreateKeyResponseBody)
	key := created.V2KeysCreateKeyResponseBody.Data
	waitForPropagation()
	t.Cleanup(func() {
		_, err := client.Keys.DeleteKey(ctx, components.V2KeysDeleteKeyRequestBody{KeyID: key.KeyID, Permanent: ptr.P(true)})
		require.NoError(t, err)
	})
	response, err := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
	require.NoError(t, err)
	require.NotNil(t, response.V2KeysGetKeyResponseBody)
	require.Len(t, response.V2KeysGetKeyResponseBody.Data.Ratelimits, 1)
	require.Equal(t, limit.Name, response.V2KeysGetKeyResponseBody.Data.Ratelimits[0].Name)
	require.Equal(t, limit.Limit, response.V2KeysGetKeyResponseBody.Data.Ratelimits[0].Limit)
	require.Equal(t, limit.Duration, response.V2KeysGetKeyResponseBody.Data.Ratelimits[0].Duration)
}
