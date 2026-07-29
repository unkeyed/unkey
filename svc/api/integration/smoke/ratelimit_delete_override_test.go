package smoke_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestDeleteOverride_RemovesOverride(t *testing.T) {
	t.Parallel()

	ctx, client := externalClient(t)
	namespace := "smoke"
	identifier := uid.DNS1035()
	_, err := client.Ratelimit.SetOverride(ctx, components.V2RatelimitSetOverrideRequestBody{
		Namespace:  namespace,
		Identifier: identifier,
		Limit:      100,
		Duration:   120_000,
	})
	require.NoError(t, err)

	var responsePresent bool
	// Override deletion may take time to propagate through regional replicas.
	require.Eventually(t, func() bool {
		response, err := client.Ratelimit.DeleteOverride(ctx, components.V2RatelimitDeleteOverrideRequestBody{
			Namespace:  namespace,
			Identifier: identifier,
		})
		responsePresent = err == nil && response.V2RatelimitDeleteOverrideResponseBody != nil
		return responsePresent
	}, 30*time.Second, time.Second)
	require.True(t, responsePresent)
}
