package server_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"

	"github.com/unkeyed/unkey/apps/agent/pkg/hash"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

func TestV1UpdateKey_UpdateAll(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := testutil.NewServer(t, resources)

	key := &authenticationv1.Key{
		KeyId:       uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.KeyAuthId,
		WorkspaceId: resources.UserWorkspace.WorkspaceId,
		Hash:        hash.Sha256(uid.New(16, "test")),
		CreatedAt:   time.Now().UnixMilli(),
	}
	err := resources.Database.InsertKey(ctx, key)
	require.NoError(t, err)

	testutil.Json[any](t, srv.App, testutil.JsonRequest{
		Method: "POST",
		Path:   "/v1/keys.updateKey",
		Body: fmt.Sprintf(`{
			"keyId": "%s",
			"name":"newName",
			"ownerId": "newOwnerId",
			"expires": null,
			"meta": {"new": "meta"},
			"ratelimit": {
				"type": "fast",
				"limit": 10,
				"refillRate": 5,
				"refillInterval": 1000
			},
			"remaining": 0
		}`, key.KeyId),
		Bearer:     resources.UserRootKey,
		StatusCode: 200,
	})

	foundKey, found, err := resources.Database.FindKeyById(ctx, key.KeyId)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "newName", *foundKey.Name)
	require.Equal(t, "newOwnerId", *foundKey.OwnerId)
	require.Equal(t, "{\"new\":\"meta\"}", *foundKey.Meta)
	require.Equal(t, authenticationv1.RatelimitType_RATELIMIT_TYPE_FAST, foundKey.Ratelimit.Type)
	require.Equal(t, int32(10), foundKey.Ratelimit.Limit)
	require.Equal(t, int32(5), foundKey.Ratelimit.RefillRate)
	require.Equal(t, int32(1000), foundKey.Ratelimit.RefillInterval)
	require.Equal(t, int32(0), *foundKey.Remaining)

}

func TestV1UpdateKey_UpdateOnlyRatelimit(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := testutil.NewServer(t, resources)

	key := &authenticationv1.Key{
		KeyId:       uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.KeyAuthId,
		WorkspaceId: resources.UserWorkspace.WorkspaceId,
		Hash:        hash.Sha256(uid.New(16, "test")),
		CreatedAt:   time.Now().UnixMilli(),
		Name:        util.Pointer("original"),
	}
	err := resources.Database.InsertKey(ctx, key)
	require.NoError(t, err)

	testutil.Json[any](t, srv.App, testutil.JsonRequest{

		Path:       "/v1/keys.updateKey",
		Body:       fmt.Sprintf(`{"keyId": "%s", "ratelimit": {"type": "fast","limit": 10,"refillRate": 5,"refillInterval": 1000}}`, key.KeyId),
		Bearer:     resources.UserRootKey,
		StatusCode: 200,
	})

	foundKey, found, err := resources.Database.FindKeyById(ctx, key.KeyId)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, key.Name, foundKey.Name)
	require.Equal(t, key.OwnerId, foundKey.OwnerId)
	require.Equal(t, key.Meta, foundKey.Meta)
	require.Equal(t, authenticationv1.RatelimitType_RATELIMIT_TYPE_FAST, foundKey.Ratelimit.Type)
	require.Equal(t, int32(10), foundKey.Ratelimit.Limit)
	require.Equal(t, int32(5), foundKey.Ratelimit.RefillRate)
	require.Equal(t, int32(1000), foundKey.Ratelimit.RefillInterval)
	require.Nil(t, foundKey.Remaining)

}

func TestV1UpdateKey_DeleteExpires(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := testutil.NewServer(t, resources)

	key := &authenticationv1.Key{
		KeyId:       uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.KeyAuthId,
		WorkspaceId: resources.UserWorkspace.WorkspaceId,
		Hash:        hash.Sha256(uid.New(16, "test")),
		CreatedAt:   time.Now().UnixMilli(),
		Expires:     util.Pointer(time.Now().Add(time.Hour).UnixMilli()),
	}

	err := resources.Database.InsertKey(ctx, key)
	require.NoError(t, err)

	testutil.Json[any](t, srv.App, testutil.JsonRequest{

		Path:       "/v1/keys.updateKey",
		Body:       fmt.Sprintf(`{"keyId": "%s", "expires": null}`, key.KeyId),
		Bearer:     resources.UserRootKey,
		StatusCode: 200,
	})

	foundKey, found, err := resources.Database.FindKeyById(ctx, key.KeyId)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, key.Name, foundKey.Name)
	require.Equal(t, key.OwnerId, foundKey.OwnerId)
	require.Equal(t, key.Meta, foundKey.Meta)
	require.Nil(t, foundKey.Expires)

}

func TestV1UpdateKey_DeleteRemaining(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := testutil.NewServer(t, resources)

	key := &authenticationv1.Key{
		KeyId:       uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.KeyAuthId,
		WorkspaceId: resources.UserWorkspace.WorkspaceId,
		Hash:        hash.Sha256(uid.New(16, "test")),
		CreatedAt:   time.Now().UnixMilli(),
	}
	ten := int32(10)
	key.Remaining = &ten

	err := resources.Database.InsertKey(ctx, key)
	require.NoError(t, err)

	testutil.Json[any](t, srv.App, testutil.JsonRequest{

		Path:       "/v1/keys.updateKey",
		Body:       fmt.Sprintf(`{"keyId": "%s", "remaining": null}`, key.KeyId),
		Bearer:     resources.UserRootKey,
		StatusCode: 200,
	})

	foundKey, found, err := resources.Database.FindKeyById(ctx, key.KeyId)
	require.NoError(t, err)
	require.True(t, found)

	require.Equal(t, key.Name, foundKey.Name)
	require.Equal(t, key.OwnerId, foundKey.OwnerId)
	require.Equal(t, key.Meta, foundKey.Meta)
	require.Nil(t, foundKey.Remaining)

}

func TestV1UpdateKey_UpdateShouldNotAffectUndefinedFields(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := testutil.NewServer(t, resources)
	key := &authenticationv1.Key{
		KeyId:       uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.KeyAuthId,
		WorkspaceId: resources.UserWorkspace.WorkspaceId,
		Hash:        hash.Sha256(uid.New(16, "test")),
		CreatedAt:   time.Now().UnixMilli(),
		Name:        util.Pointer("name"),
		OwnerId:     util.Pointer("ownerId"),
		Expires:     util.Pointer(time.Now().Add(time.Hour).UnixMilli()),
	}
	err := resources.Database.InsertKey(ctx, key)
	require.NoError(t, err)

	testutil.Json[any](t, srv.App, testutil.JsonRequest{
		Method: "POST",
		Path:   "/v1/keys.updateKey",
		Body: fmt.Sprintf(`{
			"keyId": "%s",
			"ownerId": "newOwnerId"
		}`, key.KeyId),
		Bearer:     resources.UserRootKey,
		StatusCode: 200,
	})

	foundKey, found, err := resources.Database.FindKeyById(ctx, key.KeyId)
	require.NoError(t, err)
	require.True(t, found)
	require.EqualValues(t, key.Name, foundKey.Name)
	require.Equal(t, "newOwnerId", *foundKey.OwnerId)
	require.EqualValues(t, key.Meta, foundKey.Meta)
	require.EqualValues(t, key.Ratelimit, foundKey.Ratelimit)
	require.Nil(t, foundKey.Remaining)

}
