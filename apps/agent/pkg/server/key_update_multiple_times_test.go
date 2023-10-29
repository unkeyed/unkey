package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"

	"github.com/unkeyed/unkey/apps/agent/pkg/events"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/keys"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

// Reproduction of https://github.com/unkeyed/unkey/issues/227
func TestUpdateKey_UpdateMultipleTimes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := logging.NewNoop()

	resources := testutil.SetupResources(t)

	keyCache := cache.NewMemory[*authenticationv1.Key](cache.Config[*authenticationv1.Key]{
		Fresh:   time.Minute * 15,
		Stale:   time.Minute * 60,
		MaxSize: 1024,
		RefreshFromOrigin: func(ctx context.Context, keyHash string) (*authenticationv1.Key, bool) {
			key, found, err := resources.Database.FindKeyByHash(ctx, keyHash)
			if err != nil {
				return nil, false
			}
			return key, found
		},
		Logger: logger,
	})

	apiCache := cache.NewMemory[*apisv1.Api](cache.Config[*apisv1.Api]{
		Fresh:   time.Minute * 5,
		Stale:   time.Minute * 15,
		MaxSize: 1024,
		RefreshFromOrigin: func(ctx context.Context, apiId string) (*apisv1.Api, bool) {
			key, found, err := resources.Database.FindApi(ctx, apiId)
			if err != nil {
				return nil, false
			}
			return key, found
		},
		Logger: logger,
	})
	srv := New(Config{
		Logger:   logger,
		KeyCache: keyCache,
		ApiCache: apiCache,
		Database: resources.Database,
		Tracer:   tracing.NewNoop(),
		KeyService: keys.New(keys.Config{
			Database: resources.Database,
			Events:   events.NewNoop(),
		}),
	})

	// Step 1: Create key with owner

	createKeyRequest := httptest.NewRequest("POST", "/v1/keys", bytes.NewBufferString(fmt.Sprintf(`{
		"apiId": "%s",
		"ownerId": "test_owner"
	}`, resources.UserApi.ApiId)))
	createKeyRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UserRootKey))
	createKeyRequest.Header.Set("Content-Type", "application/json")

	createKeyResponse, err := srv.app.Test(createKeyRequest)
	require.NoError(t, err)
	defer createKeyResponse.Body.Close()
	require.Equal(t, createKeyResponse.StatusCode, 200)

	createdKey := CreateKeyResponse{}
	err = json.NewDecoder(createKeyResponse.Body).Decode(&createdKey)
	require.NoError(t, err)

	foundKey, found, err := resources.Database.FindKeyById(ctx, createdKey.KeyId)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "test_owner", *foundKey.OwnerId)

	// Step 2: Update ownerId to null

	removeOwnerIdRequest := httptest.NewRequest("PUT", fmt.Sprintf("/v1/keys/%s", createdKey.KeyId), bytes.NewBufferString(`{
		"ownerId": null
	}`))
	removeOwnerIdRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UserRootKey))
	removeOwnerIdRequest.Header.Set("Content-Type", "application/json")

	removeOwnerIdResponse, err := srv.app.Test(removeOwnerIdRequest)
	require.NoError(t, err)
	defer removeOwnerIdResponse.Body.Close()
	require.Equal(t, 200, removeOwnerIdResponse.StatusCode)

	foundKeyAfterRemovingOwnerId, found, err := resources.Database.FindKeyById(ctx, createdKey.KeyId)
	require.NoError(t, err)
	require.True(t, found)
	require.Nil(t, foundKeyAfterRemovingOwnerId.OwnerId)

	// Step 3: Add a name to the key

	updateNameRequest := httptest.NewRequest("PUT", fmt.Sprintf("/v1/keys/%s", createdKey.KeyId), bytes.NewBufferString(`{
		"name": "test_name"
	}`))
	updateNameRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UserRootKey))
	updateNameRequest.Header.Set("Content-Type", "application/json")

	updateNameResponse, err := srv.app.Test(updateNameRequest)
	require.NoError(t, err)
	defer updateNameResponse.Body.Close()

	foundKeyAfterUpdatingName, found, err := resources.Database.FindKeyById(ctx, createdKey.KeyId)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "test_name", *foundKeyAfterUpdatingName.Name)
	// The ownerId should still be empty
	require.Nil(t, foundKeyAfterUpdatingName.OwnerId)
}

// unkey_3ZK8htHnoACeYjWkLHGcB3bH
