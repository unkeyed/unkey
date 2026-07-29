package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestWhoami_ReturnsKeyConfiguration(t *testing.T) {
	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	identity := createIdentity(t, ctx, client)
	name := uid.DNS1035()
	meta := map[string]any{"smokeTest": uid.DNS1035()}
	created, err := client.Keys.CreateKey(ctx, components.V2KeysCreateKeyRequestBody{APIID: api.APIID, Name: &name, ExternalID: &identity.ExternalID, Meta: meta, Enabled: ptr.P(true)})
	require.NoError(t, err)
	require.NotNil(t, created.V2KeysCreateKeyResponseBody)
	key := created.V2KeysCreateKeyResponseBody.Data
	t.Cleanup(func() {
		_, err := client.Keys.DeleteKey(ctx, components.V2KeysDeleteKeyRequestBody{KeyID: key.KeyID, Permanent: ptr.P(true)})
		require.NoError(t, err)
	})
	response, err := client.Keys.Whoami(ctx, components.V2KeysWhoamiRequestBody{Key: key.Key})
	require.NoError(t, err)
	require.NotNil(t, response.V2KeysWhoamiResponseBody)
	data := response.V2KeysWhoamiResponseBody.Data
	require.Equal(t, key.KeyID, data.KeyID)
	require.True(t, data.Enabled)
	require.NotNil(t, data.Name)
	require.Equal(t, name, *data.Name)
	require.Equal(t, meta, data.Meta)
	require.NotNil(t, data.Identity)
	require.Equal(t, identity.ExternalID, data.Identity.ExternalID)
}
