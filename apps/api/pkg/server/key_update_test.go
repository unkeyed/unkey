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

func TestUpdateKey_UpdateName(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	db, err := database.New(database.Config{
		PrimaryUs: os.Getenv("DATABASE_DSN"),
		Logger:    logging.New(),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.New(),
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
	newName := "updatedName"
	buf := bytes.NewBufferString(fmt.Sprintf(`{
		"name":"%s"
		}`, newName))

	req := httptest.NewRequest("PUT", fmt.Sprintf("/v1/keys/%s", key.Id), buf)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UnkeyKey))
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, res.StatusCode, 200)

	found, err := db.GetKeyById(ctx, key.Id)
	require.NoError(t, err)
	require.Equal(t, newName, found.Name)
}


func TestUpdateKey_RemoveName(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	db, err := database.New(database.Config{
		PrimaryUs: os.Getenv("DATABASE_DSN"),
		Logger:    logging.New(),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.New(),
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
		"name":"null"
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
	require.Equal(t, "", found.Name)
}
