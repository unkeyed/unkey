package smoke_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestMultiLimit_ReturnsEveryLimit(t *testing.T) {
	ctx, client := externalClient(t)
	response, err := client.Ratelimit.MultiLimit(ctx, []components.V2RatelimitLimitRequestBody{
		{
			Namespace:  "smoke",
			Identifier: uid.DNS1035(),
			Limit:      10,
			Duration:   60_000,
		},
		{
			Namespace:  "smoke",
			Identifier: uid.DNS1035(),
			Limit:      20,
			Duration:   60_000,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, response.V2RatelimitMultiLimitResponseBody)
	require.True(t, response.V2RatelimitMultiLimitResponseBody.Data.Passed)
	require.Len(t, response.V2RatelimitMultiLimitResponseBody.Data.Limits, 2)
}
