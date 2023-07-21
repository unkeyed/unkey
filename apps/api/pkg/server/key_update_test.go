package server

import (
	"bytes"
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/api/pkg/cache"
	"github.com/unkeyed/unkey/apps/api/pkg/database"
	"github.com/unkeyed/unkey/apps/api/pkg/entities"
	"github.com/unkeyed/unkey/apps/api/pkg/hash"
	"github.com/unkeyed/unkey/apps/api/pkg/logging"
	"github.com/unkeyed/unkey/apps/api/pkg/testutil"
	"github.com/unkeyed/unkey/apps/api/pkg/tracing"
	"github.com/unkeyed/unkey/apps/api/pkg/uid"
)

func TestUpdateKey_UpdateAll(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	db, err := database.New(database.Config{
		PrimaryUs: os.Getenv("DATABASE_DSN"),
		Logger:    logging.NewNoopLogger(),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		KeyCache: cache.NewNoopCache[entities.Key](),
		ApiCache: cache.NewNoopCache[entities.Api](),
		Database: db,
		Tracer:   tracing.NewNoop(),
	})

	key := entities.Key{
		Id:          uid.Key(),
		ApiId:       resources.UserApi.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(uid.New(16, "test")),
		CreatedAt:   time.Now(),
	}
	err = db.CreateKey(ctx, key)
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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UnkeyKey))
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()



	require.Equal(t, res.StatusCode, 200)

	found, err := db.GetKeyById(ctx, key.Id)
	require.NoError(t, err)
	require.Equal(t, "newName", found.Name)
	require.Equal(t, "newOwnerId", found.OwnerId)
	require.Equal(t, map[string]any{"new": "meta"}, found.Meta)
	require.Equal(t, "fast", found.Ratelimit.Type)
	require.Equal(t, int64(10), found.Ratelimit.Limit)
	require.Equal(t, int64(5), found.Ratelimit.RefillRate)
	require.Equal(t, int64(1000), found.Ratelimit.RefillInterval)
	require.Equal(t, int64(0), found.Remaining.Remaining)

}

func TestUpdateKey_UpdateOnlyRatelimit(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	db, err := database.New(database.Config{
		PrimaryUs: os.Getenv("DATABASE_DSN"),
		Logger:    logging.NewNoopLogger(),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		KeyCache: cache.NewNoopCache[entities.Key](),
		ApiCache: cache.NewNoopCache[entities.Api](),
		Database: db,
		Tracer:   tracing.NewNoop(),
	})

	key := entities.Key{
		Id:          uid.Key(),
		ApiId:       resources.UserApi.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(uid.New(16, "test")),
		CreatedAt:   time.Now(),
		Name:        "original",
	}
	err = db.CreateKey(ctx, key)
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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UnkeyKey))
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()



	require.Equal(t, res.StatusCode, 200)

	found, err := db.GetKeyById(ctx, key.Id)
	require.NoError(t, err)
	require.Equal(t, key.Name, found.Name)
	require.Equal(t, key.OwnerId, found.OwnerId)
	require.Equal(t, key.Meta, found.Meta)
	require.Equal(t, "fast", found.Ratelimit.Type)
	require.Equal(t, int64(10), found.Ratelimit.Limit)
	require.Equal(t, int64(5), found.Ratelimit.RefillRate)
	require.Equal(t, int64(1000), found.Ratelimit.RefillInterval)
	require.Equal(t, key.Remaining.Remaining, found.Remaining.Remaining)

}

func TestUpdateKey_DeleteExpires(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	db, err := database.New(database.Config{
		PrimaryUs: os.Getenv("DATABASE_DSN"),
		Logger:    logging.NewNoopLogger(),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		KeyCache: cache.NewNoopCache[entities.Key](),
		ApiCache: cache.NewNoopCache[entities.Api](),
		Database: db,
		Tracer:   tracing.NewNoop(),
	})

	key := entities.Key{
		Id:          uid.Key(),
		ApiId:       resources.UserApi.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(uid.New(16, "test")),
		CreatedAt:   time.Now(),
		Expires:     time.Now().Add(time.Hour),
	}

	err = db.CreateKey(ctx, key)
	require.NoError(t, err)
	buf := bytes.NewBufferString(`{
		"expires": null
	}`)

	req := httptest.NewRequest("PUT", fmt.Sprintf("/v1/keys/%s", key.Id), buf)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UnkeyKey))
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()



	require.Equal(t, res.StatusCode, 200)

	found, err := db.GetKeyById(ctx, key.Id)
	require.NoError(t, err)
	require.Equal(t, key.Name, found.Name)
	require.Equal(t, key.OwnerId, found.OwnerId)
	require.Equal(t, key.Meta, found.Meta)
	require.True(t, found.Expires.IsZero())

}

func TestUpdateKey_DeleteRemaining(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	db, err := database.New(database.Config{
		PrimaryUs: os.Getenv("DATABASE_DSN"),
		Logger:    logging.NewNoopLogger(),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		KeyCache: cache.NewNoopCache[entities.Key](),
		ApiCache: cache.NewNoopCache[entities.Api](),
		Database: db,
		Tracer:   tracing.NewNoop(),
	})

	key := entities.Key{
		Id:          uid.Key(),
		ApiId:       resources.UserApi.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(uid.New(16, "test")),
		CreatedAt:   time.Now(),
	}
	key.Remaining.Enabled = true
	key.Remaining.Remaining = 10

	err = db.CreateKey(ctx, key)
	require.NoError(t, err)
	buf := bytes.NewBufferString(`{
		"remaining": null
	}`)

	req := httptest.NewRequest("PUT", fmt.Sprintf("/v1/keys/%s", key.Id), buf)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UnkeyKey))
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()



	require.Equal(t, res.StatusCode, 200)

	found, err := db.GetKeyById(ctx, key.Id)
	require.NoError(t, err)
	require.Equal(t, key.Name, found.Name)
	require.Equal(t, key.OwnerId, found.OwnerId)
	require.Equal(t, key.Meta, found.Meta)
	require.Equal(t, false, found.Remaining.Enabled)
	require.Equal(t, int64(0), found.Remaining.Remaining)

}

func TestUpdateKey_UpdateShouldNotAffectUndefinedFields(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	db, err := database.New(database.Config{
		PrimaryUs: os.Getenv("DATABASE_DSN"),
		Logger:    logging.NewNoopLogger(),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		KeyCache: cache.NewNoopCache[entities.Key](),
		ApiCache: cache.NewNoopCache[entities.Api](),
		Database: db,
		Tracer:   tracing.NewNoop(),
	})

	key := entities.Key{
		Id:          uid.Key(),
		ApiId:       resources.UserApi.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(uid.New(16, "test")),
		CreatedAt:   time.Now(),
		Name:        "name",
		OwnerId:     "ownerId",
		Expires:     time.Now().Add(time.Hour),
	}
	err = db.CreateKey(ctx, key)
	require.NoError(t, err)
	buf := bytes.NewBufferString(`{
		"ownerId": "newOwnerId"
	}`)

	req := httptest.NewRequest("PUT", fmt.Sprintf("/v1/keys/%s", key.Id), buf)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UnkeyKey))
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()



	require.Equal(t, res.StatusCode, 200)

	found, err := db.GetKeyById(ctx, key.Id)
	require.NoError(t, err)
	require.Equal(t, key.Name, found.Name)
	require.Equal(t, "newOwnerId", found.OwnerId)
	require.Equal(t, key.Meta, found.Meta)
	require.Equal(t, key.Ratelimit, found.Ratelimit)
	require.Equal(t, key.Remaining.Remaining, found.Remaining.Remaining)

}
