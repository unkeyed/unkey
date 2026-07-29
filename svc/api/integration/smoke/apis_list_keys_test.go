package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func TestListKeys_ReturnsCreatedKey(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	api := createAPI(t, ctx, client)
	key := createKey(t, ctx, client, api.APIID)
	response, err := client.Apis.ListKeys(ctx, components.V2ApisListKeysRequestBody{APIID: api.APIID, Limit: ptr.P(int64(10)), RevalidateKeysCache: ptr.P(true)})
	require.NoError(t, err)
	require.NotNil(t, response.V2ApisListKeysResponseBody)
	require.Contains(t, keyIDs(response.V2ApisListKeysResponseBody.Data), key.KeyID)
}
