package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestUpdateKey_PersistsNameAndMetadata(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	key := createKey(t, ctx, client, api.APIID)
	name := uid.DNS1035()
	meta := map[string]any{"smokeTest": uid.DNS1035()}
	response, err := client.Keys.UpdateKey(ctx, components.V2KeysUpdateKeyRequestBody{KeyID: key.KeyID, Name: &name, Meta: meta})
	require.NoError(t, err)
	require.NotNil(t, response.V2KeysUpdateKeyResponseBody)
	get, err := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
	require.NoError(t, err)
	require.NotNil(t, get.V2KeysGetKeyResponseBody)
	require.NotNil(t, get.V2KeysGetKeyResponseBody.Data.Name)
	require.Equal(t, name, *get.V2KeysGetKeyResponseBody.Data.Name)
	require.Equal(t, meta, get.V2KeysGetKeyResponseBody.Data.Meta)
}

func TestUpdateKey_PersistsIdentityAssociation(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	key := createKey(t, ctx, client, api.APIID)
	identity := createIdentity(t, ctx, client)
	_, err := client.Keys.UpdateKey(ctx, components.V2KeysUpdateKeyRequestBody{KeyID: key.KeyID, ExternalID: &identity.ExternalID})
	require.NoError(t, err)
	get, err := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
	require.NoError(t, err)
	require.NotNil(t, get.V2KeysGetKeyResponseBody.Data.Identity)
	require.Equal(t, identity.ExternalID, get.V2KeysGetKeyResponseBody.Data.Identity.ExternalID)
}

func TestUpdateKey_PersistsRatelimit(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	key := createKey(t, ctx, client, api.APIID)
	limit := components.RatelimitRequest{Name: uid.DNS1035(), Limit: 15, Duration: 60_000, AutoApply: ptr.P(true)}
	_, err := client.Keys.UpdateKey(ctx, components.V2KeysUpdateKeyRequestBody{KeyID: key.KeyID, Ratelimits: []components.RatelimitRequest{limit}})
	require.NoError(t, err)
	get, err := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
	require.NoError(t, err)
	require.NotNil(t, get.V2KeysGetKeyResponseBody)
	require.Len(t, get.V2KeysGetKeyResponseBody.Data.Ratelimits, 1)
	require.Equal(t, limit.Name, get.V2KeysGetKeyResponseBody.Data.Ratelimits[0].Name)
	require.Equal(t, limit.Limit, get.V2KeysGetKeyResponseBody.Data.Ratelimits[0].Limit)
}
