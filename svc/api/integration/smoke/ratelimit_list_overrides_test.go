package smoke_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestListOverrides_ReturnsPersistedOverride(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	namespace := "smoke"
	identifier := uid.DNS1035()
	set, err := client.Ratelimit.SetOverride(ctx, components.V2RatelimitSetOverrideRequestBody{
		Namespace:  namespace,
		Identifier: identifier,
		Limit:      100,
		Duration:   120_000,
	})
	require.NoError(t, err)
	require.NotNil(t, set.V2RatelimitSetOverrideResponseBody)
	t.Cleanup(func() {
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			_, err := client.Ratelimit.DeleteOverride(ctx, components.V2RatelimitDeleteOverrideRequestBody{
				Namespace:  namespace,
				Identifier: identifier,
			})
			require.NoError(c, err)
		}, 30*time.Second, time.Second)
	})
	// Override listings may take time to propagate through regional replicas.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		response, err := client.Ratelimit.ListOverrides(ctx, components.V2RatelimitListOverridesRequestBody{
			Namespace: namespace,
			Limit:     ptr.P(int64(100)),
		})
		require.NoError(c, err)
		require.NotNil(c, response.V2RatelimitListOverridesResponseBody)
		found := false
		for _, override := range response.V2RatelimitListOverridesResponseBody.Data {
			if override.OverrideID == set.V2RatelimitSetOverrideResponseBody.Data.OverrideID {
				found = true
				break
			}
		}
		require.True(c, found, "override %q was not listed", set.V2RatelimitSetOverrideResponseBody.Data.OverrideID)
	}, 30*time.Second, time.Second)
}
