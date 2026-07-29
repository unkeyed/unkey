package smoke_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		get, getErr := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
		require.NoError(c, getErr)
		require.NotNil(c, get.V2KeysGetKeyResponseBody)
		require.Equal(c, &name, get.V2KeysGetKeyResponseBody.Data.Name)
		require.Equal(c, meta, get.V2KeysGetKeyResponseBody.Data.Meta)
	}, 30*time.Second, time.Second)
}

func TestUpdateKey_PersistsIdentityAssociation(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	key := createKey(t, ctx, client, api.APIID)
	identity := createIdentity(t, ctx, client)
	_, err := client.Keys.UpdateKey(ctx, components.V2KeysUpdateKeyRequestBody{KeyID: key.KeyID, ExternalID: &identity.ExternalID})
	require.NoError(t, err)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		get, getErr := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
		require.NoError(c, getErr)
		require.NotNil(c, get.V2KeysGetKeyResponseBody)
		require.NotNil(c, get.V2KeysGetKeyResponseBody.Data.Identity)
		require.Equal(c, identity.ExternalID, get.V2KeysGetKeyResponseBody.Data.Identity.ExternalID)
	}, 30*time.Second, time.Second)
}

func TestUpdateKey_PersistsRatelimit(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	key := createKey(t, ctx, client, api.APIID)
	limit := components.RatelimitRequest{Name: uid.DNS1035(), Limit: 15, Duration: 60_000, AutoApply: ptr.P(true)}
	_, err := client.Keys.UpdateKey(ctx, components.V2KeysUpdateKeyRequestBody{KeyID: key.KeyID, Ratelimits: []components.RatelimitRequest{limit}})
	require.NoError(t, err)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		get, getErr := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
		require.NoError(c, getErr)
		require.NotNil(c, get.V2KeysGetKeyResponseBody)
		require.Len(c, get.V2KeysGetKeyResponseBody.Data.Ratelimits, 1)
		persisted := get.V2KeysGetKeyResponseBody.Data.Ratelimits[0]
		require.Equal(c, limit.Name, persisted.Name)
		require.Equal(c, limit.Limit, persisted.Limit)
		require.Equal(c, limit.Duration, persisted.Duration)
		require.Equal(c, limit.AutoApply, persisted.AutoApply)
	}, 30*time.Second, time.Second)
}
