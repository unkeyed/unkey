package smoke_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func TestUpdateCredits_SetsRemainingCredits(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	key := createKey(t, ctx, client, api.APIID)
	remaining := int64(10)

	response, err := client.Keys.UpdateCredits(ctx, components.V2KeysUpdateCreditsRequestBody{
		KeyID:     key.KeyID,
		Operation: components.OperationSet,
		Value:     &remaining,
	})
	require.NoError(t, err)
	require.NotNil(t, response.V2KeysUpdateCreditsResponseBody)
	require.Equal(t, &remaining, response.V2KeysUpdateCreditsResponseBody.Data.Remaining)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		getResponse, getErr := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
		require.NoError(c, getErr)
		require.NotNil(c, getResponse.V2KeysGetKeyResponseBody)
		require.NotNil(c, getResponse.V2KeysGetKeyResponseBody.Data.Credits)
		require.Equal(c, &remaining, getResponse.V2KeysGetKeyResponseBody.Data.Credits.Remaining)
	}, 30*time.Second, time.Second)
}

func TestUpdateCredits_IncrementsRemainingCredits(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	key := createKey(t, ctx, client, api.APIID)
	initial := int64(10)
	increment := int64(5)

	_, err := client.Keys.UpdateCredits(ctx, components.V2KeysUpdateCreditsRequestBody{
		KeyID:     key.KeyID,
		Operation: components.OperationSet,
		Value:     &initial,
	})
	require.NoError(t, err)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		getResponse, getErr := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
		require.NoError(c, getErr)
		require.NotNil(c, getResponse.V2KeysGetKeyResponseBody)
		require.NotNil(c, getResponse.V2KeysGetKeyResponseBody.Data.Credits)
		require.Equal(c, &initial, getResponse.V2KeysGetKeyResponseBody.Data.Credits.Remaining)
	}, 30*time.Second, time.Second)
	response, err := client.Keys.UpdateCredits(ctx, components.V2KeysUpdateCreditsRequestBody{
		KeyID:     key.KeyID,
		Operation: components.OperationIncrement,
		Value:     &increment,
	})
	require.NoError(t, err)
	require.NotNil(t, response.V2KeysUpdateCreditsResponseBody)
	require.Equal(t, ptr.P(int64(15)), response.V2KeysUpdateCreditsResponseBody.Data.Remaining)
}

func TestUpdateCredits_DecrementsRemainingCredits(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	key := createKey(t, ctx, client, api.APIID)
	initial := int64(10)
	decrement := int64(4)

	_, err := client.Keys.UpdateCredits(ctx, components.V2KeysUpdateCreditsRequestBody{
		KeyID:     key.KeyID,
		Operation: components.OperationSet,
		Value:     &initial,
	})
	require.NoError(t, err)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		getResponse, getErr := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
		require.NoError(c, getErr)
		require.NotNil(c, getResponse.V2KeysGetKeyResponseBody)
		require.NotNil(c, getResponse.V2KeysGetKeyResponseBody.Data.Credits)
		require.Equal(c, &initial, getResponse.V2KeysGetKeyResponseBody.Data.Credits.Remaining)
	}, 30*time.Second, time.Second)
	response, err := client.Keys.UpdateCredits(ctx, components.V2KeysUpdateCreditsRequestBody{
		KeyID:     key.KeyID,
		Operation: components.OperationDecrement,
		Value:     &decrement,
	})
	require.NoError(t, err)
	require.NotNil(t, response.V2KeysUpdateCreditsResponseBody)
	require.Equal(t, ptr.P(int64(6)), response.V2KeysUpdateCreditsResponseBody.Data.Remaining)
}

func TestUpdateCredits_CanMakeKeyUnlimited(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	key := createKey(t, ctx, client, api.APIID)
	initial := int64(10)

	_, err := client.Keys.UpdateCredits(ctx, components.V2KeysUpdateCreditsRequestBody{
		KeyID:     key.KeyID,
		Operation: components.OperationSet,
		Value:     &initial,
	})
	require.NoError(t, err)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		getResponse, getErr := client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{KeyID: key.KeyID})
		require.NoError(c, getErr)
		require.NotNil(c, getResponse.V2KeysGetKeyResponseBody)
		require.NotNil(c, getResponse.V2KeysGetKeyResponseBody.Data.Credits)
		require.Equal(c, &initial, getResponse.V2KeysGetKeyResponseBody.Data.Credits.Remaining)
	}, 30*time.Second, time.Second)
	response, err := client.Keys.UpdateCredits(ctx, components.V2KeysUpdateCreditsRequestBody{
		KeyID:     key.KeyID,
		Operation: components.OperationSet,
		Value:     nil,
	})
	require.NoError(t, err)
	require.NotNil(t, response.V2KeysUpdateCreditsResponseBody)
	require.Nil(t, response.V2KeysUpdateCreditsResponseBody.Data.Remaining)
}
