package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestLimit_DecrementsRemainingTokens(t *testing.T) {
	ctx, client := externalClient(t)
	response, err := client.Ratelimit.Limit(ctx, components.V2RatelimitLimitRequestBody{
		Namespace:  "smoke",
		Identifier: uid.DNS1035(),
		Limit:      10,
		Duration:   60_000,
	})
	require.NoError(t, err)
	require.NotNil(t, response.V2RatelimitLimitResponseBody)
	require.True(t, response.V2RatelimitLimitResponseBody.Data.Success)
	require.Equal(t, int64(10), response.V2RatelimitLimitResponseBody.Data.Limit)
	require.Equal(t, int64(9), response.V2RatelimitLimitResponseBody.Data.Remaining)
}
