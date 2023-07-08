package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/unkeyed/unkey/apps/api/pkg/cache"
	"github.com/unkeyed/unkey/apps/api/pkg/database"
	"github.com/unkeyed/unkey/apps/api/pkg/entities"
	"github.com/unkeyed/unkey/apps/api/pkg/hash"
	"github.com/unkeyed/unkey/apps/api/pkg/logging"
	"github.com/unkeyed/unkey/apps/api/pkg/testutil"
	"github.com/unkeyed/unkey/apps/api/pkg/tracing"
	"github.com/unkeyed/unkey/apps/api/pkg/uid"
	"github.com/stretchr/testify/require"
)

func TestListKeys_Simple(t *testing.T) {
	ctx := context.Background()
	resources := testutil.SetupResources(t)

	db, err := database.New(database.Config{
		Logger: logging.NewNoopLogger(),

		PrimaryUs: os.Getenv("DATABASE_DSN"),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		Cache:    cache.NewNoopCache[entities.Key](),
		Database: db,
		Tracer:   tracing.NewNoop(),
	})

	createdKeyIds := make([]string, 10)
	for i := range createdKeyIds {
		key := entities.Key{
			Id:          uid.Key(),
			ApiId:       resources.UserApi.Id,
			WorkspaceId: resources.UserWorkspace.Id,
			Hash:        hash.Sha256(uid.New(16, "test")),
			CreatedAt:   time.Now(),
		}
		err := db.CreateKey(ctx, key)
		require.NoError(t, err)
		createdKeyIds[i] = key.Id

	}

	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/apis/%s/keys", resources.UserApi.Id), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UnkeyKey))

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, 200, res.StatusCode)

	successResponse := ListKeysResponse{}
	err = json.Unmarshal(body, &successResponse)
	require.NoError(t, err)

	require.GreaterOrEqual(t, successResponse.Total, len(createdKeyIds))
	require.GreaterOrEqual(t, len(successResponse.Keys), len(createdKeyIds))
	require.LessOrEqual(t, len(successResponse.Keys), 100) //  default page size

}

func TestListKeys_FilterOwnerId(t *testing.T) {
	ctx := context.Background()
	resources := testutil.SetupResources(t)

	db, err := database.New(database.Config{
		Logger: logging.NewNoopLogger(),

		PrimaryUs: os.Getenv("DATABASE_DSN"),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		Cache:    cache.NewNoopCache[entities.Key](),
		Database: db,
		Tracer:   tracing.NewNoop(),
	})

	createdKeyIds := make([]string, 10)
	for i := range createdKeyIds {
		key := entities.Key{
			Id:          uid.Key(),
			ApiId:       resources.UserApi.Id,
			WorkspaceId: resources.UserWorkspace.Id,
			Hash:        hash.Sha256(uid.New(16, "test")),
			CreatedAt:   time.Now(),
		}
		// just add an ownerId to half of them
		if i%2 == 0 {
			key.OwnerId = "chronark"
		}
		err := db.CreateKey(ctx, key)
		require.NoError(t, err)
		createdKeyIds[i] = key.Id

	}

	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/apis/%s/keys?ownerId=chronark", resources.UserApi.Id), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UnkeyKey))

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, 200, res.StatusCode)

	successResponse := ListKeysResponse{}
	err = json.Unmarshal(body, &successResponse)
	require.NoError(t, err)

	require.GreaterOrEqual(t, successResponse.Total, len(createdKeyIds))
	require.Equal(t, 5, len(successResponse.Keys))
	require.LessOrEqual(t, len(successResponse.Keys), 100) //  default page size

	for _, key := range successResponse.Keys {
		require.Equal(t, "chronark", key.OwnerId)
	}

}

func TestListKeys_WithLimit(t *testing.T) {
	ctx := context.Background()
	resources := testutil.SetupResources(t)

	db, err := database.New(database.Config{
		Logger: logging.NewNoopLogger(),

		PrimaryUs: os.Getenv("DATABASE_DSN"),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		Cache:    cache.NewNoopCache[entities.Key](),
		Database: db,
		Tracer:   tracing.NewNoop(),
	})

	createdKeyIds := make([]string, 10)
	for i := range createdKeyIds {
		key := entities.Key{
			Id:          uid.Key(),
			ApiId:       resources.UserApi.Id,
			WorkspaceId: resources.UserWorkspace.Id,
			Hash:        hash.Sha256(uid.New(16, "test")),
			CreatedAt:   time.Now(),
		}
		err := db.CreateKey(ctx, key)
		require.NoError(t, err)
		createdKeyIds[i] = key.Id

	}

	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/apis/%s/keys?limit=2", resources.UserApi.Id), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UnkeyKey))

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)

	successResponse := ListKeysResponse{}
	err = json.Unmarshal(body, &successResponse)
	require.NoError(t, err)

	require.GreaterOrEqual(t, successResponse.Total, len(createdKeyIds))
	require.Equal(t, 2, len(successResponse.Keys))

}

func TestListKeys_WithOffset(t *testing.T) {
	ctx := context.Background()
	resources := testutil.SetupResources(t)

	db, err := database.New(database.Config{
		Logger: logging.NewNoopLogger(),

		PrimaryUs: os.Getenv("DATABASE_DSN"),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.New(),
		Cache:    cache.NewNoopCache[entities.Key](),
		Database: db,
		Tracer:   tracing.NewNoop(),
	})

	createdKeyIds := make([]string, 10)
	for i := range createdKeyIds {
		key := entities.Key{
			Id:          uid.Key(),
			ApiId:       resources.UserApi.Id,
			WorkspaceId: resources.UserWorkspace.Id,
			Hash:        hash.Sha256(uid.New(16, "test")),
			CreatedAt:   time.Now(),
		}
		err := db.CreateKey(ctx, key)
		require.NoError(t, err)
		createdKeyIds[i] = key.Id

	}

	req1 := httptest.NewRequest("GET", fmt.Sprintf("/v1/apis/%s/keys", resources.UserApi.Id), nil)
	req1.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UnkeyKey))

	res1, err := srv.app.Test(req1)
	require.NoError(t, err)
	defer res1.Body.Close()

	body1, err := io.ReadAll(res1.Body)
	require.NoError(t, err)
	require.Equal(t, 200, res1.StatusCode)

	successResponse1 := ListKeysResponse{}
	err = json.Unmarshal(body1, &successResponse1)
	require.NoError(t, err)

	require.GreaterOrEqual(t, successResponse1.Total, len(createdKeyIds))
	require.GreaterOrEqual(t, 10, len(successResponse1.Keys))

	req2 := httptest.NewRequest("GET", fmt.Sprintf("/v1/apis/%s/keys?offset=1", resources.UserApi.Id), nil)
	req2.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UnkeyKey))

	res2, err := srv.app.Test(req2)
	require.NoError(t, err)
	defer res2.Body.Close()

	body2, err := io.ReadAll(res2.Body)
	require.NoError(t, err)
	require.Equal(t, 200, res2.StatusCode)

	successResponse2 := ListKeysResponse{}
	err = json.Unmarshal(body2, &successResponse2)
	require.NoError(t, err)

	require.GreaterOrEqual(t, successResponse2.Total, len(createdKeyIds))
	require.GreaterOrEqual(t, len(successResponse2.Keys), len(createdKeyIds)-2)

	require.Equal(t, successResponse1.Keys[1].Id, successResponse2.Keys[0].Id)

}
