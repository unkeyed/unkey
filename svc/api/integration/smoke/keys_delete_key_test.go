package smoke_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
)

func TestDeleteKey_InvalidatesVerification(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	created, err := client.Keys.CreateKey(ctx, components.V2KeysCreateKeyRequestBody{APIID: api.APIID, Enabled: new(true)})
	require.NoError(t, err)
	require.NotNil(t, created.V2KeysCreateKeyResponseBody)
	key := created.V2KeysCreateKeyResponseBody.Data
	waitForPropagation()
	deleted, err := client.Keys.DeleteKey(ctx, components.V2KeysDeleteKeyRequestBody{KeyID: key.KeyID, Permanent: new(true)})
	require.NoError(t, err)
	require.NotNil(t, deleted.V2KeysDeleteKeyResponseBody)
	// The delete commits on the primary while verification reads a replica, so
	// assert the state the API converges to instead of assuming zero lag.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		verified, verifyErr := client.Keys.VerifyKey(ctx, components.V2KeysVerifyKeyRequestBody{Key: key.Key})
		require.NoError(c, verifyErr)
		require.NotNil(c, verified.V2KeysVerifyKeyResponseBody)
		require.False(c, verified.V2KeysVerifyKeyResponseBody.Data.Valid)
	}, 30*time.Second, time.Second)
}
