package server

import (
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/chronark/unkey/apps/api/pkg/cache"
	"github.com/chronark/unkey/apps/api/pkg/database"
	"github.com/chronark/unkey/apps/api/pkg/entities"
	"github.com/chronark/unkey/apps/api/pkg/hash"
	"github.com/chronark/unkey/apps/api/pkg/logging"
	"github.com/chronark/unkey/apps/api/pkg/testutil"
	"github.com/chronark/unkey/apps/api/pkg/tracing"

	"github.com/chronark/unkey/apps/api/pkg/uid"
	"github.com/stretchr/testify/require"
)

func TestDeleteKey(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	db, err := database.New(database.Config{
		PrimaryUs: os.Getenv("DATABASE_DSN"),
		Tracer:    tracing.NewNoop(),
	})
	require.NoError(t, err)

	key := entities.Key{
		Id:          uid.Key(),
		ApiId:       resources.UserApi.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(uid.New(16, "test")),
		CreatedAt:   time.Now(),
	}
	err = db.CreateKey(ctx, key)
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		Cache:    cache.NewInMemoryCache[entities.Key](cache.Config{Ttl: time.Minute, Tracer: tracing.NewNoop()}),
		Database: db,
		Tracer:   tracing.NewNoop(),
	})

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/v1/keys/%s", key.Id), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UnkeyKey))

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	_, err = io.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, 200, res.StatusCode)

	_, err = db.GetKeyById(ctx, key.Id)
	require.Error(t, err)
}
