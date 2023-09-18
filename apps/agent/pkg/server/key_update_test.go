package server

import (
	"bytes"
	"context"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/hash"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

func TestUpdateKey_UpdateAll(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		KeyCache: cache.NewNoopCache[entities.Key](),
		ApiCache: cache.NewNoopCache[entities.Api](),
		Database: resources.Database,
		Tracer:   tracing.NewNoop(),
	})

	key := entities.Key{
		Id:          uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(uid.New(16, "test")),
		CreatedAt:   time.Now(),
	}
	err := resources.Database.InsertKey(ctx, key)
	require.NoError(t, err)
	buf := bytes.NewBufferString(`{
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
	}`)

	req := httptest.NewRequest("PUT", fmt.Sprintf("/v1/keys/%s", key.Id), buf)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UserRootKey))
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, res.StatusCode, 200)

	foundKey, found, err := resources.Database.FindKeyById(ctx, key.Id)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "newName", foundKey.Name)
	require.Equal(t, "newOwnerId", foundKey.OwnerId)
	require.Equal(t, map[string]any{"new": "meta"}, foundKey.Meta)
	require.Equal(t, "fast", foundKey.Ratelimit.Type)
	require.Equal(t, int32(10), foundKey.Ratelimit.Limit)
	require.Equal(t, int32(5), foundKey.Ratelimit.RefillRate)
	require.Equal(t, int32(1000), foundKey.Ratelimit.RefillInterval)
	require.Equal(t, int32(0), *foundKey.Remaining)

}

func TestUpdateKey_UpdateOnlyRatelimit(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		KeyCache: cache.NewNoopCache[entities.Key](),
		ApiCache: cache.NewNoopCache[entities.Api](),
		Database: resources.Database,
		Tracer:   tracing.NewNoop(),
	})

	key := entities.Key{
		Id:          uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(uid.New(16, "test")),
		CreatedAt:   time.Now(),
		Name:        "original",
	}
	err := resources.Database.InsertKey(ctx, key)
	require.NoError(t, err)
	buf := bytes.NewBufferString(`{
		"ratelimit": {
			"type": "fast",
			"limit": 10,
			"refillRate": 5,
			"refillInterval": 1000
		}
	}`)

	req := httptest.NewRequest("PUT", fmt.Sprintf("/v1/keys/%s", key.Id), buf)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UserRootKey))
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, res.StatusCode, 200)

	foundKey, found, err := resources.Database.FindKeyById(ctx, key.Id)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, key.Name, foundKey.Name)
	require.Equal(t, key.OwnerId, foundKey.OwnerId)
	require.Equal(t, key.Meta, foundKey.Meta)
	require.Equal(t, "fast", foundKey.Ratelimit.Type)
	require.Equal(t, int32(10), foundKey.Ratelimit.Limit)
	require.Equal(t, int32(5), foundKey.Ratelimit.RefillRate)
	require.Equal(t, int32(1000), foundKey.Ratelimit.RefillInterval)
	require.Nil(t, foundKey.Remaining)

}

func TestUpdateKey_DeleteExpires(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		KeyCache: cache.NewNoopCache[entities.Key](),
		ApiCache: cache.NewNoopCache[entities.Api](),
		Database: resources.Database,
		Tracer:   tracing.NewNoop(),
	})

	key := entities.Key{
		Id:          uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(uid.New(16, "test")),
		CreatedAt:   time.Now(),
		Expires:     time.Now().Add(time.Hour),
	}

	err := resources.Database.InsertKey(ctx, key)
	require.NoError(t, err)
	buf := bytes.NewBufferString(`{
		"expires": null
	}`)

	req := httptest.NewRequest("PUT", fmt.Sprintf("/v1/keys/%s", key.Id), buf)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UserRootKey))
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, res.StatusCode, 200)

	foundKey, found, err := resources.Database.FindKeyById(ctx, key.Id)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, key.Name, foundKey.Name)
	require.Equal(t, key.OwnerId, foundKey.OwnerId)
	require.Equal(t, key.Meta, foundKey.Meta)
	require.True(t, foundKey.Expires.IsZero())

}

func TestUpdateKey_DeleteRemaining(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		KeyCache: cache.NewNoopCache[entities.Key](),
		ApiCache: cache.NewNoopCache[entities.Api](),
		Database: resources.Database,
		Tracer:   tracing.NewNoop(),
	})

	key := entities.Key{
		Id:          uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(uid.New(16, "test")),
		CreatedAt:   time.Now(),
	}
	ten := int32(10)
	key.Remaining = &ten

	err := resources.Database.InsertKey(ctx, key)
	require.NoError(t, err)
	buf := bytes.NewBufferString(`{
		"remaining": null
	}`)

	req := httptest.NewRequest("PUT", fmt.Sprintf("/v1/keys/%s", key.Id), buf)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UserRootKey))
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, res.StatusCode, 200)

	foundKey, found, err := resources.Database.FindKeyById(ctx, key.Id)
	require.NoError(t, err)
	require.True(t, found)

	require.Equal(t, key.Name, foundKey.Name)
	require.Equal(t, key.OwnerId, foundKey.OwnerId)
	require.Equal(t, key.Meta, foundKey.Meta)
	require.Nil(t, foundKey.Remaining)

}

func TestUpdateKey_UpdateShouldNotAffectUndefinedFields(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		KeyCache: cache.NewNoopCache[entities.Key](),
		ApiCache: cache.NewNoopCache[entities.Api](),
		Database: resources.Database,
		Tracer:   tracing.NewNoop(),
	})

	key := entities.Key{
		Id:          uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(uid.New(16, "test")),
		CreatedAt:   time.Now(),
		Name:        "name",
		OwnerId:     "ownerId",
		Expires:     time.Now().Add(time.Hour),
	}
	err := resources.Database.InsertKey(ctx, key)
	require.NoError(t, err)
	buf := bytes.NewBufferString(`{
		"ownerId": "newOwnerId"
	}`)

	req := httptest.NewRequest("PUT", fmt.Sprintf("/v1/keys/%s", key.Id), buf)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UserRootKey))
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, res.StatusCode, 200)

	foundKey, found, err := resources.Database.FindKeyById(ctx, key.Id)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, key.Name, foundKey.Name)
	require.Equal(t, "newOwnerId", foundKey.OwnerId)
	require.Equal(t, key.Meta, foundKey.Meta)
	require.Equal(t, key.Ratelimit, foundKey.Ratelimit)
	require.Nil(t, foundKey.Remaining)

}
