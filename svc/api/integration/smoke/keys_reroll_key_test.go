package smoke_test

import (
	"testing"

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

	verification, err := client.Keys.VerifyKey(ctx, components.V2KeysVerifyKeyRequestBody{Key: rerolled.Key})
	require.NoError(t, err)
	require.NotNil(t, verification.V2KeysVerifyKeyResponseBody)
	require.True(t, verification.V2KeysVerifyKeyResponseBody.Data.Valid)
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

	getResponse, err := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: rerolled.KeyID})
	require.NoError(t, err)
	require.NotNil(t, getResponse.V2KeysGetKeyResponseBody)
	persisted := getResponse.V2KeysGetKeyResponseBody.Data
	require.Equal(t, &name, persisted.Name)
	require.Equal(t, metadata, persisted.Meta)
	require.NotNil(t, persisted.Identity)
	require.Equal(t, identity.ExternalID, persisted.Identity.ExternalID)
	require.Len(t, persisted.Ratelimits, 1)
	require.Equal(t, ratelimit.Name, persisted.Ratelimits[0].Name)
}
