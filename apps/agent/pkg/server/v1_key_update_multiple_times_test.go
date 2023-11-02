package server_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/server"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
)

// Reproduction of https://github.com/unkeyed/unkey/issues/227
func V1TestUpdateKey_UpdateMultipleTimes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := testutil.NewServer(t, resources)

	// Step 1: Create key with owner
	createdKey := testutil.Json[server.CreateKeyResponse](t, srv.App, testutil.JsonRequest{

		Path:       "/v1/keys",
		Bearer:     resources.UserRootKey,
		Body:       fmt.Sprintf(`{"apiId":"%s", "ownerId": "test_owner"}`, resources.UserApi.ApiId),
		StatusCode: 200,
	})

	foundKey, found, err := resources.Database.FindKeyById(ctx, createdKey.KeyId)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "test_owner", *foundKey.OwnerId)

	// Step 2: Update ownerId to null

	testutil.Json[server.UpdateKeyResponse](t, srv.App, testutil.JsonRequest{

		Path:       "/v1/keys.updateKey",
		Bearer:     resources.UserRootKey,
		Body:       `{"ownerId": null}`,
		StatusCode: 200,
	})

	foundKeyAfterRemovingOwnerId, found, err := resources.Database.FindKeyById(ctx, createdKey.KeyId)
	require.NoError(t, err)
	require.True(t, found)
	require.Nil(t, foundKeyAfterRemovingOwnerId.OwnerId)

	// Step 3: Add a name to the key

	testutil.Json[server.UpdateKeyResponse](t, srv.App, testutil.JsonRequest{
		Method:     "PUT",
		Path:       fmt.Sprintf("/v1/keys/%s", createdKey.KeyId),
		Bearer:     resources.UserRootKey,
		Body:       `{"name": "test_name"}`,
		StatusCode: 200,
	})

	foundKeyAfterUpdatingName, found, err := resources.Database.FindKeyById(ctx, createdKey.KeyId)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "test_name", *foundKeyAfterUpdatingName.Name)
	// The ownerId should still be empty
	require.Nil(t, foundKeyAfterUpdatingName.OwnerId)
}
