package identities

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	unkey "github.com/unkeyed/unkey-go"
	"github.com/unkeyed/unkey-go/models/components"
	"github.com/unkeyed/unkey-go/models/operations"
	attack "github.com/unkeyed/unkey/apps/agent/pkg/testutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

func TestIdentitiesRatelimitAccuracy(t *testing.T) {
	// Step 1 --------------------------------------------------------------------
	// Setup the sdk, create an API and an identity
	// ---------------------------------------------------------------------------

	ctx := context.Background()
	rootKey := os.Getenv("INTEGRATION_TEST_ROOT_KEY")
	require.NotEmpty(t, rootKey, "INTEGRATION_TEST_ROOT_KEY must be set")
	baseURL := os.Getenv("UNKEY_BASE_URL")
	require.NotEmpty(t, baseURL, "UNKEY_BASE_URL must be set")

	sdk := unkey.New(
		unkey.WithServerURL(baseURL),
		unkey.WithSecurity(rootKey),
	)

	for _, nKeys := range []int{1, 3, 10, 1000} {
		t.Run(fmt.Sprintf("with %d keys", nKeys), func(t *testing.T) {

			for _, tc := range []struct {
				rate         attack.Rate
				testDuration time.Duration
			}{
				{
					rate:         attack.Rate{Freq: 20, Per: time.Second},
					testDuration: 1 * time.Minute,
				},
				{
					rate:         attack.Rate{Freq: 100, Per: time.Second},
					testDuration: 5 * time.Minute,
				},
			} {
				t.Run(fmt.Sprintf("[%s] over %s", tc.rate.String(), tc.testDuration), func(t *testing.T) {
					api, err := sdk.Apis.CreateAPI(ctx, operations.CreateAPIRequestBody{
						Name: uid.New("testapi"),
					})
					require.NoError(t, err)

					externalId := uid.New("testuser")

					_, err = sdk.Identities.CreateIdentity(ctx, operations.CreateIdentityRequestBody{
						ExternalID: externalId,
						Meta: map[string]any{
							"email": "test@test.com",
						},
					})
					require.NoError(t, err)

					// Step 2 --------------------------------------------------------------------
					// Update the identity with ratelimits
					// ---------------------------------------------------------------------------

					inferenceLimit := operations.UpdateIdentityRatelimits{
						Name:     "inferenceLimit",
						Limit:    100,
						Duration: time.Minute.Milliseconds(),
					}

					_, err = sdk.Identities.UpdateIdentity(ctx, operations.UpdateIdentityRequestBody{
						ExternalID: unkey.String(externalId),
						Ratelimits: []operations.UpdateIdentityRatelimits{inferenceLimit},
					})
					require.NoError(t, err)

					// Step 4 --------------------------------------------------------------------
					// Create keys that share the same identity and therefore the same ratelimits
					// ---------------------------------------------------------------------------

					keys := make([]operations.CreateKeyResponseBody, nKeys)
					for i := 0; i < len(keys); i++ {
						key, err := sdk.Keys.CreateKey(ctx, operations.CreateKeyRequestBody{
							APIID:       api.Object.APIID,
							ExternalID:  unkey.String(externalId),
							Environment: unkey.String("integration_test"),
						})
						require.NoError(t, err)
						keys[i] = *key.Object
					}

					// Step 5 --------------------------------------------------------------------
					// Test ratelimits
					// ---------------------------------------------------------------------------

					total := 0
					passed := 0

					results := attack.Attack(t, tc.rate, tc.testDuration, func() bool {

						// Each request uses one of the keys randomly
						key := util.RandomElement(keys).Key

						res, err := sdk.Keys.VerifyKey(context.Background(), components.V1KeysVerifyKeyRequest{
							APIID: unkey.String(api.Object.APIID),
							Key:   key,
							Ratelimits: []components.Ratelimits{
								{Name: inferenceLimit.Name},
							},
						})
						require.NoError(t, err)

						return res.V1KeysVerifyKeyResponse.Valid

					})

					for valid := range results {
						total++
						if valid {
							passed++
						}

					}

					// Step 6 --------------------------------------------------------------------
					// Assert ratelimits worked
					// ---------------------------------------------------------------------------

					exactLimit := int(inferenceLimit.Limit) * int(tc.testDuration/(time.Duration(inferenceLimit.Duration)*time.Millisecond))
					upperLimit := int(2.5 * float64(exactLimit))
					lowerLimit := exactLimit
					if total < lowerLimit {
						lowerLimit = total
					}
					t.Logf("Total: %d, Passed: %d, lowerLimit: %d, exactLimit: %d, upperLimit: %d", total, passed, lowerLimit, exactLimit, upperLimit)

					// check requests::api is not exceeded
					require.GreaterOrEqual(t, passed, lowerLimit)
					require.LessOrEqual(t, passed, upperLimit)
				})
			}
		})
	}
}
