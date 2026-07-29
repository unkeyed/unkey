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

func TestRerollKey_ReturnsNewWorkingKey(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	original := createKey(t, ctx, client, api.APIID)

	response, err := client.Keys.RerollKey(ctx, components.V2KeysRerollKeyRequestBody{
		KeyID:      original.KeyID,
		Expiration: 0,
	})
	require.NoError(t, err)
	require.NotNil(t, response.V2KeysRerollKeyResponseBody)
	rerolled := response.V2KeysRerollKeyResponseBody.Data
	require.NotEmpty(t, rerolled.KeyID)
	require.NotEmpty(t, rerolled.Key)
	require.NotEqual(t, original.KeyID, rerolled.KeyID)
	require.NotEqual(t, original.Key, rerolled.Key)
	t.Cleanup(func() {
		_, err := client.Keys.DeleteKey(ctx, components.V2KeysDeleteKeyRequestBody{
			KeyID:     rerolled.KeyID,
			Permanent: ptr.P(true),
		})
		require.NoError(t, err)
	})

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		verification, verifyErr := client.Keys.VerifyKey(ctx, components.V2KeysVerifyKeyRequestBody{Key: rerolled.Key})
		require.NoError(c, verifyErr)
		require.NotNil(c, verification.V2KeysVerifyKeyResponseBody)
		require.True(c, verification.V2KeysVerifyKeyResponseBody.Data.Valid)
	}, 30*time.Second, time.Second)
}

func TestRerollKey_PreservesConfiguration(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	identity := createIdentity(t, ctx, client)
	name := uid.DNS1035()
	metadata := map[string]any{"smokeTest": uid.DNS1035()}
	ratelimit := components.RatelimitRequest{
		Name:      uid.DNS1035(),
		Limit:     10,
		Duration:  60_000,
		AutoApply: ptr.P(true),
	}

	createResponse, err := client.Keys.CreateKey(ctx, components.V2KeysCreateKeyRequestBody{
		APIID:      api.APIID,
		Name:       &name,
		ExternalID: &identity.ExternalID,
		Meta:       metadata,
		Ratelimits: []components.RatelimitRequest{ratelimit},
	})
	require.NoError(t, err)
	require.NotNil(t, createResponse.V2KeysCreateKeyResponseBody)
	original := createResponse.V2KeysCreateKeyResponseBody.Data
	waitForPropagation()

	response, err := client.Keys.RerollKey(ctx, components.V2KeysRerollKeyRequestBody{
		KeyID:      original.KeyID,
		Expiration: 0,
	})
	require.NoError(t, err)
	require.NotNil(t, response.V2KeysRerollKeyResponseBody)
	rerolled := response.V2KeysRerollKeyResponseBody.Data

	t.Cleanup(func() {
		_, err := client.Keys.DeleteKey(ctx, components.V2KeysDeleteKeyRequestBody{
			KeyID:     rerolled.KeyID,
			Permanent: ptr.P(true),
		})
		require.NoError(t, err)
	})

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		getResponse, getErr := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: rerolled.KeyID})
		require.NoError(c, getErr)
		require.NotNil(c, getResponse.V2KeysGetKeyResponseBody)
		persisted := getResponse.V2KeysGetKeyResponseBody.Data
		require.Equal(c, &name, persisted.Name)
		require.Equal(c, metadata, persisted.Meta)
		require.NotNil(c, persisted.Identity)
		require.Equal(c, identity.ExternalID, persisted.Identity.ExternalID)
		require.Len(c, persisted.Ratelimits, 1)
		require.Equal(c, ratelimit.Name, persisted.Ratelimits[0].Name)
		require.Equal(c, ratelimit.Limit, persisted.Ratelimits[0].Limit)
		require.Equal(c, ratelimit.Duration, persisted.Ratelimits[0].Duration)
		require.Equal(c, ratelimit.AutoApply, persisted.Ratelimits[0].AutoApply)
	}, 30*time.Second, time.Second)
}
