package keys_test

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

func TestDefaultRatelimitAccuracy(t *testing.T) {
	// Step 1 --------------------------------------------------------------------
	// Setup the sdk, create an API and a key
	// ---------------------------------------------------------------------------

	ctx := context.Background()
	rootKey := os.Getenv("INTEGRATION_TEST_ROOT_KEY")
	require.NotEmpty(t, rootKey, "INTEGRATION_TEST_ROOT_KEY must be set")
	baseURL := os.Getenv("UNKEY_BASE_URL")
	require.NotEmpty(t, baseURL, "UNKEY_BASE_URL must be set")

	options := []unkey.SDKOption{
		unkey.WithSecurity(rootKey),
	}

	if baseURL != "" {
		options = append(options, unkey.WithServerURL(baseURL))
	}
	sdk := unkey.New(options...)

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

			// Step 2 --------------------------------------------------------------------
			// Update the identity with ratelimits
			// ---------------------------------------------------------------------------

			// Step 3 --------------------------------------------------------------------
			// Create keys that share the same identity and therefore the same ratelimits
			// ---------------------------------------------------------------------------

			ratelimit := operations.Ratelimit{
				Limit:    100,
				Duration: util.Pointer(time.Minute.Milliseconds()),
			}

			key, err := sdk.Keys.CreateKey(ctx, operations.CreateKeyRequestBody{
				APIID:     api.Object.APIID,
				Ratelimit: &ratelimit,
			})
			require.NoError(t, err)

			// Step 5 --------------------------------------------------------------------
			// Test ratelimits
			// ---------------------------------------------------------------------------

			total := 0
			passed := 0

			results := attack.Attack(t, tc.rate, tc.testDuration, func() bool {

				res, err := sdk.Keys.VerifyKey(context.Background(), components.V1KeysVerifyKeyRequest{
					APIID: unkey.String(api.Object.APIID),
					Key:   key.Object.Key,
					Ratelimits: []components.Ratelimits{
						{Name: "default"},
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

			exactLimit := int(ratelimit.Limit) * int(tc.testDuration/(time.Duration(*ratelimit.Duration)*time.Millisecond))
			upperLimit := int(1.2 * float64(exactLimit))
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
}
