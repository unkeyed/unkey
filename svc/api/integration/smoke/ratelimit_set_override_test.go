package smoke_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestSetOverride_AppliesOverride(t *testing.T) {
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
		require.Eventually(t, func() bool {
			_, err := client.Ratelimit.DeleteOverride(ctx, components.V2RatelimitDeleteOverrideRequestBody{
				Namespace:  namespace,
				Identifier: identifier,
			})
			return err == nil
		}, 30*time.Second, time.Second)
	})
	// Override enforcement may take time to propagate through regional caches.
	require.Eventually(t, func() bool {
		limited, err := client.Ratelimit.Limit(ctx, components.V2RatelimitLimitRequestBody{
			Namespace:  namespace,
			Identifier: identifier,
			Limit:      10,
			Duration:   60_000,
		})
		return err == nil && limited.V2RatelimitLimitResponseBody != nil && limited.V2RatelimitLimitResponseBody.Data.Limit == override.Limit && limited.V2RatelimitLimitResponseBody.Data.OverrideID != nil && *limited.V2RatelimitLimitResponseBody.Data.OverrideID == overrideID
	}, 30*time.Second, time.Second)
}
