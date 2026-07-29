package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func TestDeleteKey_InvalidatesVerification(t *testing.T) {
	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	created, err := client.Keys.CreateKey(ctx, components.V2KeysCreateKeyRequestBody{APIID: api.APIID, Enabled: ptr.P(true)})
	require.NoError(t, err)
	require.NotNil(t, created.V2KeysCreateKeyResponseBody)
	key := created.V2KeysCreateKeyResponseBody.Data
	waitForPropagation()
	deleted, err := client.Keys.DeleteKey(ctx, components.V2KeysDeleteKeyRequestBody{KeyID: key.KeyID, Permanent: ptr.P(true)})
	require.NoError(t, err)
	require.NotNil(t, deleted.V2KeysDeleteKeyResponseBody)
	verified, err := client.Keys.VerifyKey(ctx, components.V2KeysVerifyKeyRequestBody{Key: key.Key})
	require.NoError(t, err)
	require.NotNil(t, verified.V2KeysVerifyKeyResponseBody)
	require.False(t, verified.V2KeysVerifyKeyResponseBody.Data.Valid)
}
