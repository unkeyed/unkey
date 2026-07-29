package smoke_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestGetOverride_ReturnsPersistedOverride(t *testing.T) {
	ctx, client := externalClient(t)
	namespace := "smoke"
	identifier := uid.DNS1035()
	override := components.V2RatelimitSetOverrideRequestBody{
		Namespace:  namespace,
		Identifier: identifier,
		Limit:      100,
		Duration:   120_000,
	}
	set, err := client.Ratelimit.SetOverride(ctx, override)
	require.NoError(t, err)
	require.NotNil(t, set.V2RatelimitSetOverrideResponseBody)
	t.Cleanup(func() {
		require.Eventually(t, func() bool {
			_, err := client.Ratelimit.DeleteOverride(ctx, components.V2RatelimitDeleteOverrideRequestBody{
				Namespace:  namespace,
				Identifier: identifier,
			})
			return err == nil
		}, 30*time.Second, time.Second)
	})

	var persisted components.RatelimitOverride
	// Override reads may take time to propagate through regional replicas.
	require.Eventually(t, func() bool {
		response, err := client.Ratelimit.GetOverride(ctx, components.V2RatelimitGetOverrideRequestBody{
			Namespace:  namespace,
			Identifier: identifier,
		})
		if err != nil || response.V2RatelimitGetOverrideResponseBody == nil {
			return false
		}
		persisted = response.V2RatelimitGetOverrideResponseBody.Data
		return true
	}, 30*time.Second, time.Second)
	require.Equal(t, set.V2RatelimitSetOverrideResponseBody.Data.OverrideID, persisted.OverrideID)
	require.Equal(t, override.Identifier, persisted.Identifier)
	require.Equal(t, override.Limit, persisted.Limit)
	require.Equal(t, override.Duration, persisted.Duration)
}
