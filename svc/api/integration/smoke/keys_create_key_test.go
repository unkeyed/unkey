package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func TestCreateKey_ReturnsFetchableKey(t *testing.T) {
	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	response, err := client.Keys.CreateKey(ctx, components.V2KeysCreateKeyRequestBody{APIID: api.APIID, Enabled: ptr.P(true)})
	require.NoError(t, err)
	require.NotNil(t, response.V2KeysCreateKeyResponseBody)
	key := response.V2KeysCreateKeyResponseBody.Data
	require.NotEmpty(t, key.KeyID)
	require.NotEmpty(t, key.Key)
	t.Cleanup(func() {
		_, err := client.Keys.DeleteKey(ctx, components.V2KeysDeleteKeyRequestBody{KeyID: key.KeyID, Permanent: ptr.P(true)})
		require.NoError(t, err)
	})
	get, err := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
	require.NoError(t, err)
	require.NotNil(t, get.V2KeysGetKeyResponseBody)
	require.Equal(t, key.KeyID, get.V2KeysGetKeyResponseBody.Data.KeyID)
}
