package identities

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	unkey "github.com/unkeyed/unkey-go"
	"github.com/unkeyed/unkey-go/models/components"
	"github.com/unkeyed/unkey-go/models/operations"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

func TestClusterRatelimitAccuracy(t *testing.T) {
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

	_, err = sdk.Identities.UpdateIdentity(ctx, operations.UpdateIdentityRequestBody{
		ExternalID: unkey.String(externalId),
		Ratelimits: []operations.UpdateIdentityRatelimits{

			// A baseline defense against requests of all kind
			{
				Name:     "requests::api",
				Limit:    100,
				Duration: time.Second.Milliseconds(),
			},

			// llama-v3p1-405b-instruct
			{
				// Limit the number of requests to 100 per minute
				Name:     "requests::llama-v3p1-405b-instruct",
				Limit:    100,
				Duration: time.Minute.Milliseconds(),
			},
			{
				// Limit the number of tokens consumed to 100k per hour
				Name:     "tokens::llama-v3p1-405b-instruct",
				Limit:    100_000,
				Duration: (time.Minute).Milliseconds(),
			},

			// mixtral-8x22b-instruct
			// Assuming this one is cheaper, we can set higher limits
			{
				// Limit the number of requests to 1000 per minute
				Name:     "requests::mixtral-8x22b-instruct",
				Limit:    1000,
				Duration: time.Minute.Milliseconds(),
			},
			{
				// Limit the number of tokens consumed to 20mil per hour
				Name:     "tokens::mixtral-8x22b-instruct",
				Limit:    20_000_000,
				Duration: (time.Minute).Milliseconds(),
			},
		},
	})
	require.NoError(t, err)

	// Step 4 --------------------------------------------------------------------
	// Create keys that share the same identity and therefore the same ratelimits
	// ---------------------------------------------------------------------------

	keys := make([]operations.CreateKeyResponseBody, 5)
	for i := 0; i < 5; i++ {
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

	// Collect the results
	total := 0
	totalPassed := 0
	llamaRequestsPassed := 0
	llamaTokensPassed := 0
	mixtralRequestsPassed := 0
	mixtralTokensPassed := 0

	seconds := 5 * 60 // how long the test should run
	rps := 50         // how many requests per second we make

	resultsC := make(chan testResult, rps)

	go func() {
		for res := range resultsC {
			total++
			if res.valid {
				totalPassed++
				switch res.model {
				case "llama-v3p1-405b-instruct":
					llamaRequestsPassed++
					llamaTokensPassed += int(res.tokens)
				case "mixtral-8x22b-instruct":
					mixtralRequestsPassed++
					mixtralTokensPassed += int(res.tokens)
				}
			}

		}
	}()

	var wg sync.WaitGroup

	// Make requests

	for range seconds {
		time.Sleep(time.Second)
		for range rps {
			wg.Add(1)
			go func() {

				// Each request uses one of the keys, a random model and a random number of tokens
				key := util.RandomElement(keys).Key
				model := util.RandomElement([]string{"llama-v3p1-405b-instruct", "mixtral-8x22b-instruct"})
				tokens := 1000 + rand.Int63n(5000)

				res, err := sdk.Keys.VerifyKey(context.Background(), components.V1KeysVerifyKeyRequest{
					APIID: unkey.String(api.Object.APIID),
					Key:   key,
					Ratelimits: []components.Ratelimits{
						// Check baseline
						{Name: "requests::api"},
						// Check requests against model
						{Name: fmt.Sprintf("requests::%s", model)},
						// Check tokens against model
						{Name: fmt.Sprintf("tokens::%s", model), Cost: unkey.Int64(tokens)},
					},
				})
				require.NoError(t, err)

				resultsC <- testResult{model, tokens, res.V1KeysVerifyKeyResponse.Valid}

				wg.Done()
			}()
		}
	}

	wg.Wait()

	// Step 6 --------------------------------------------------------------------
	// Assert ratelimits worked
	// ---------------------------------------------------------------------------

	t.Logf("Total: %d, Passed: %d", total, totalPassed)
	t.Logf("Llama Requests: %d, Tokens: %d", llamaRequestsPassed, llamaTokensPassed)
	t.Logf("Mixtral Requests: %d, Tokens: %d", mixtralRequestsPassed, mixtralTokensPassed)

	// check requests::api is not exceeded
	require.LessOrEqual(t, totalPassed, int(100*float64(seconds)*1.2))

	// check requests::llama-v3p1-405b-instruct is not exceeded
	require.LessOrEqual(t, llamaRequestsPassed, int(float64(seconds)*100/60*1.2))

	// check tokens::llama-v3p1-405b-instruct is not exceeded
	require.LessOrEqual(t, llamaTokensPassed, int(float64(seconds)*100_000/60*1.2))

	// check requests::mixtral-8x22b-instruct is not exceeded
	require.LessOrEqual(t, mixtralRequestsPassed, int(float64(seconds)*100_000/60*1.2))

	// check tokens::mixtral-8x22b-instruct is not exceeded
	require.LessOrEqual(t, mixtralTokensPassed, int(float64(seconds)*20_000_000/60*1.2))
}

type testResult struct {
	model  string
	tokens int64
	valid  bool
}
