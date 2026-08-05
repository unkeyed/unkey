package smoke_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestSetOverride_AppliesOverride(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	namespace := "smoke"
	identifier := uid.DNS1035()
	override := components.V2RatelimitSetOverrideRequestBody{
		Namespace:  namespace,
		Identifier: identifier,
		Limit:      100,
		Duration:   120_000,
	}
	response, err := client.Ratelimit.SetOverride(ctx, override)
	require.NoError(t, err)
	require.NotNil(t, response.V2RatelimitSetOverrideResponseBody)
	overrideID := response.V2RatelimitSetOverrideResponseBody.Data.OverrideID
	t.Cleanup(func() {
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			_, err := client.Ratelimit.DeleteOverride(ctx, components.V2RatelimitDeleteOverrideRequestBody{
				Namespace:  namespace,
				Identifier: identifier,
			})
			require.NoError(c, err)
		}, 30*time.Second, time.Second)
	})
	// Override enforcement may take time to propagate through regional caches.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		limited, err := client.Ratelimit.Limit(ctx, components.V2RatelimitLimitRequestBody{
			Namespace:  namespace,
			Identifier: identifier,
			Limit:      10,
			Duration:   60_000,
		})
		require.NoError(c, err)
		require.NotNil(c, limited.V2RatelimitLimitResponseBody)
		require.Equal(c, override.Limit, limited.V2RatelimitLimitResponseBody.Data.Limit)
		require.NotNil(c, limited.V2RatelimitLimitResponseBody.Data.OverrideID)
		require.Equal(c, overrideID, *limited.V2RatelimitLimitResponseBody.Data.OverrideID)
	}, 30*time.Second, time.Second)
}
