package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/chronark/unkey/apps/api/pkg/cache"
	"github.com/chronark/unkey/apps/api/pkg/database"
	"github.com/chronark/unkey/apps/api/pkg/entities"
	"github.com/chronark/unkey/apps/api/pkg/tracing"

	"github.com/chronark/unkey/apps/api/pkg/hash"
	"github.com/chronark/unkey/apps/api/pkg/logging"
	"github.com/chronark/unkey/apps/api/pkg/testutil"
	"github.com/chronark/unkey/apps/api/pkg/uid"
	"github.com/stretchr/testify/require"
)

func TestVerifyKey_Simple(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	db, err := database.New(database.Config{
		PrimaryUs: os.Getenv("DATABASE_DSN"),
		Tracer:    tracing.NewNoop(),
	})
	require.NoError(t, err)

	key := uid.New(16, "test")
	err = db.CreateKey(ctx, entities.Key{
		Id:          uid.Key(),
		ApiId:       resources.UserApi.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(key),
		CreatedAt:   time.Now(),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		Cache:    cache.NewInMemoryCache[entities.Key](cache.Config{Ttl: time.Minute, Tracer: tracing.NewNoop()}),
		Database: db,
		Tracer:   tracing.NewNoop(),
	})

	buf := bytes.NewBufferString(fmt.Sprintf(`{
		"key":"%s"
		}`, key))

	req := httptest.NewRequest("POST", "/v1/keys/verify", buf)
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)

	successResponse := VerifyKeyResponse{}
	err = json.Unmarshal(body, &successResponse)
	require.NoError(t, err)

	require.True(t, successResponse.Valid)

}

func TestVerifyKey_WithTemporaryKey(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	db, err := database.New(database.Config{
		PrimaryUs: os.Getenv("DATABASE_DSN"),
		Tracer:    tracing.NewNoop(),
	})
	require.NoError(t, err)

	key := uid.New(16, "test")
	err = db.CreateKey(ctx, entities.Key{
		Id:          uid.Key(),
		ApiId:       resources.UserApi.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(key),
		CreatedAt:   time.Now(),
		Expires:     time.Now().Add(time.Second * 5),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		Cache:    cache.NewInMemoryCache[entities.Key](cache.Config{Ttl: time.Minute, Tracer: tracing.NewNoop()}),
		Database: db,
		Tracer:   tracing.NewNoop(),
	})

	buf := bytes.NewBufferString(fmt.Sprintf(`{
		"key":"%s"
		}`, key))

	req := httptest.NewRequest("POST", "/v1/keys/verify", buf)
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)

	successResponse := VerifyKeyResponse{}
	err = json.Unmarshal(body, &successResponse)
	require.NoError(t, err)

	require.True(t, successResponse.Valid)

	// wait until key expires
	time.Sleep(time.Second * 5)

	errorRes, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	errorBody, err := io.ReadAll(errorRes.Body)
	require.NoError(t, err)
	require.Equal(t, 404, errorRes.StatusCode)

	errorResponse := VerifyKeyErrorResponse{}
	err = json.Unmarshal(errorBody, &errorResponse)
	require.NoError(t, err)

	require.False(t, errorResponse.Valid)

}
