package identities

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	unkey "github.com/unkeyed/unkey-go"
	"github.com/unkeyed/unkey-go/models/components"
	"github.com/unkeyed/unkey-go/models/operations"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

func TestUpdateAutomaticallyCreatedIdentityWithManyKeys(t *testing.T) {

	// Step 1 --------------------------------------------------------------------
	// Setup the sdk, create an API and keys
	// ---------------------------------------------------------------------------

	ctx := context.Background()
	rootKey := os.Getenv("INTEGRATION_TEST_ROOT_KEY")
	if rootKey == "" {
		t.Skip("INTEGRATION_TEST_ROOT_KEY is not set")
	}
	baseURL := os.Getenv("UNKEY_BASE_URL")

	options := []unkey.SDKOption{
		unkey.WithSecurity(rootKey),
	}

	if baseURL != "" {
		options = append(options, unkey.WithServerURL(baseURL))
	}
	sdk := unkey.New(options...)

	api, err := sdk.Apis.CreateAPI(ctx, operations.CreateAPIRequestBody{
		Name: uid.New("testapi"),
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err := sdk.Apis.DeleteAPI(ctx, operations.DeleteAPIRequestBody{
			APIID: api.Object.APIID,
		})
		require.NoError(t, err)
	})

	externalId := uid.New("testuser")

	keys := make([]*operations.CreateKeyResponseBody, 1000)
	for i := range keys {
		key, err := sdk.Keys.CreateKey(ctx, operations.CreateKeyRequestBody{
			APIID:   api.Object.APIID,
			OwnerID: unkey.String(externalId),
		})
		require.NoError(t, err)

		t.Cleanup(func() {
			_, err := sdk.Keys.DeleteKey(ctx, operations.DeleteKeyRequestBody{
				KeyID: key.Object.KeyID,
			})
			require.NoError(t, err)
		})

		keys[i] = key.Object
	}

	// Step 2 --------------------------------------------------------------------
	// Update the identity with ratelimits
	// ---------------------------------------------------------------------------

	// Create fake ratelimits
	ratelimits := make([]operations.UpdateIdentityRatelimits, 50)
	for i := range ratelimits {
		ratelimits[i] = operations.UpdateIdentityRatelimits{
			Name:     uid.New("ratelimit"),
			Limit:    100,
			Duration: time.Second.Milliseconds(),
		}
	}

	_, err = sdk.Identities.UpdateIdentity(ctx, operations.UpdateIdentityRequestBody{
		ExternalID: unkey.String(externalId),
		Meta: map[string]any{
			"hello": "world",
		},
		Ratelimits: ratelimits,
	})
	require.NoError(t, err)

	// Step 3 --------------------------------------------------------------------
	// Verify the keys to see if they are updated
	// ---------------------------------------------------------------------------

	for _, key := range keys {
		verifyRes, err := sdk.Keys.VerifyKey(ctx, components.V1KeysVerifyKeyRequest{
			APIID: unkey.String(api.Object.APIID),
			Key:   key.Key,
		})
		require.NoError(t, err)

		require.True(t, verifyRes.V1KeysVerifyKeyResponse.Valid)
		require.NotNil(t, verifyRes.V1KeysVerifyKeyResponse.Identity)
		require.Equal(t, externalId, verifyRes.V1KeysVerifyKeyResponse.Identity.ExternalID)

		meta, err := json.Marshal(verifyRes.V1KeysVerifyKeyResponse.Identity.Meta)
		require.NoError(t, err)
		require.JSONEq(t, `{"hello":"world"}`, string(meta))
	}
}
