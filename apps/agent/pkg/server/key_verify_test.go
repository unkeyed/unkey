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

	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/database"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/hash"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

func TestVerifyKey_Simple(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	db, err := database.New(database.Config{
		Logger:    logging.NewNoopLogger(),
		PrimaryUs: os.Getenv("DATABASE_DSN"),
	})
	require.NoError(t, err)

	key := uid.New(16, "test")
	err = db.CreateKey(ctx, entities.Key{
		Id:          uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(key),
		CreatedAt:   time.Now(),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		KeyCache: cache.NewNoopCache[entities.Key](),
		ApiCache: cache.NewNoopCache[entities.Api](),
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

	t.Logf("body: %s", string(body))
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
		Logger: logging.NewNoopLogger(),

		PrimaryUs: os.Getenv("DATABASE_DSN"),
	})
	require.NoError(t, err)

	key := uid.New(16, "test")
	err = db.CreateKey(ctx, entities.Key{
		Id:          uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(key),
		CreatedAt:   time.Now(),
		Expires:     time.Now().Add(time.Second * 5),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		KeyCache: cache.NewNoopCache[entities.Key](),
		ApiCache: cache.NewNoopCache[entities.Api](),
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
	require.Equal(t, 200, errorRes.StatusCode)

	verifyKeyResponse := VerifyKeyResponse{}
	err = json.Unmarshal(errorBody, &verifyKeyResponse)
	require.NoError(t, err)

	require.False(t, verifyKeyResponse.Valid)

}

func TestVerifyKey_WithRatelimit(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	db, err := database.New(database.Config{
		Logger: logging.NewNoopLogger(),

		PrimaryUs: os.Getenv("DATABASE_DSN"),
	})
	require.NoError(t, err)

	key := uid.New(16, "test")
	err = db.CreateKey(ctx, entities.Key{
		Id:          uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(key),
		CreatedAt:   time.Now(),
		Ratelimit: &entities.Ratelimit{
			Type:           "fast",
			Limit:          2,
			RefillRate:     1,
			RefillInterval: 10000,
		},
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:    logging.NewNoopLogger(),
		KeyCache:  cache.NewNoopCache[entities.Key](),
		ApiCache:  cache.NewNoopCache[entities.Api](),
		Database:  db,
		Tracer:    tracing.NewNoop(),
		Ratelimit: ratelimit.NewInMemory(),
	})

	buf := bytes.NewBufferString(fmt.Sprintf(`{
		"key":"%s"
		}`, key))

	req := httptest.NewRequest("POST", "/v1/keys/verify", buf)
	req.Header.Set("Content-Type", "application/json")

	// first request

	res1, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res1.Body.Close()

	body1, err := io.ReadAll(res1.Body)
	require.NoError(t, err)
	require.Equal(t, 200, res1.StatusCode)

	verifyRes1 := VerifyKeyResponse{}
	err = json.Unmarshal(body1, &verifyRes1)
	require.NoError(t, err)

	require.True(t, verifyRes1.Valid)
	require.Equal(t, int32(2), verifyRes1.Ratelimit.Limit)
	require.Equal(t, int32(1), verifyRes1.Ratelimit.Remaining)
	require.GreaterOrEqual(t, verifyRes1.Ratelimit.Reset, int64(time.Now().UnixMilli()))
	require.LessOrEqual(t, verifyRes1.Ratelimit.Reset, int64(time.Now().Add(time.Second*10).UnixMilli()))

	// second request

	res2, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res2.Body.Close()

	body2, err := io.ReadAll(res2.Body)
	require.NoError(t, err)
	require.Equal(t, 200, res2.StatusCode)

	verifyRes2 := VerifyKeyResponse{}
	err = json.Unmarshal(body2, &verifyRes2)
	require.NoError(t, err)

	require.True(t, verifyRes2.Valid)
	require.Equal(t, int32(2), verifyRes2.Ratelimit.Limit)
	require.Equal(t, int32(0), verifyRes2.Ratelimit.Remaining)
	require.GreaterOrEqual(t, verifyRes2.Ratelimit.Reset, int64(time.Now().UnixMilli()))
	require.LessOrEqual(t, verifyRes2.Ratelimit.Reset, int64(time.Now().Add(time.Second*10).UnixMilli()))

	// third request

	res3, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res3.Body.Close()

	body3, err := io.ReadAll(res3.Body)
	require.NoError(t, err)
	require.Equal(t, 200, res3.StatusCode)

	verifyRes3 := VerifyKeyResponse{}
	err = json.Unmarshal(body3, &verifyRes3)
	require.NoError(t, err)

	require.False(t, verifyRes3.Valid)
	require.Equal(t, int32(2), verifyRes3.Ratelimit.Limit)
	require.Equal(t, int32(0), verifyRes3.Ratelimit.Remaining)
	require.GreaterOrEqual(t, verifyRes3.Ratelimit.Reset, int64(time.Now().UnixMilli()))
	require.LessOrEqual(t, verifyRes3.Ratelimit.Reset, int64(time.Now().Add(time.Second*10).UnixMilli()))

	// wait and try again in the next window
	time.Sleep(time.Until(time.UnixMilli(verifyRes3.Ratelimit.Reset)))

	res4, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res4.Body.Close()

	body4, err := io.ReadAll(res4.Body)
	require.NoError(t, err)
	require.Equal(t, 200, res4.StatusCode)

	verifyRes4 := VerifyKeyResponse{}
	err = json.Unmarshal(body4, &verifyRes4)
	require.NoError(t, err)

	require.True(t, verifyRes4.Valid)
	require.Equal(t, int32(2), verifyRes4.Ratelimit.Limit)
	require.Equal(t, int32(0), verifyRes4.Ratelimit.Remaining)
	require.GreaterOrEqual(t, verifyRes4.Ratelimit.Reset, int64(time.Now().UnixMilli()))
	require.LessOrEqual(t, verifyRes4.Ratelimit.Reset, int64(time.Now().Add(time.Second*10).UnixMilli()))

}

func TestVerifyKey_WithIpWhitelist_Pass(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	db, err := database.New(database.Config{
		Logger:    logging.NewNoopLogger(),
		PrimaryUs: os.Getenv("DATABASE_DSN"),
	})
	require.NoError(t, err)

	keyAuth := entities.KeyAuth{
		Id:          uid.KeyAuth(),
		WorkspaceId: resources.UserWorkspace.Id,
	}
	err = db.CreateKeyAuth(ctx, keyAuth)
	require.NoError(t, err)

	api := entities.Api{
		Id:          uid.Api(),
		Name:        "test",
		WorkspaceId: resources.UserWorkspace.Id,
		IpWhitelist: []string{"100.100.100.100"},
		AuthType:    entities.AuthTypeKey,
		KeyAuthId:   keyAuth.Id,
	}
	err = db.InsertApi(ctx, api)
	require.NoError(t, err)

	key := uid.New(16, "test")
	err = db.CreateKey(ctx, entities.Key{
		Id:          uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.Id,
		WorkspaceId: api.WorkspaceId,
		Hash:        hash.Sha256(key),
		CreatedAt:   time.Now(),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		KeyCache: cache.NewNoopCache[entities.Key](),
		ApiCache: cache.NewNoopCache[entities.Api](),
		Database: db,
		Tracer:   tracing.NewNoop(),
	})

	buf := bytes.NewBufferString(fmt.Sprintf(`{
		"key":"%s"
		}`, key))

	req := httptest.NewRequest("POST", "/v1/keys/verify", buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Fly-Client-IP", "100.100.100.100")

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

func TestVerifyKey_WithIpWhitelist_Blocked(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	db, err := database.New(database.Config{
		Logger: logging.NewNoopLogger(),

		PrimaryUs: os.Getenv("DATABASE_DSN"),
	})
	require.NoError(t, err)

	keyAuth := entities.KeyAuth{
		Id:          uid.KeyAuth(),
		WorkspaceId: resources.UserWorkspace.Id,
	}
	err = db.CreateKeyAuth(ctx, keyAuth)
	require.NoError(t, err)

	api := entities.Api{
		Id:          uid.Api(),
		KeyAuthId:   keyAuth.Id,
		AuthType:    entities.AuthTypeKey,
		Name:        "test",
		WorkspaceId: resources.UserWorkspace.Id,
		IpWhitelist: []string{"100.100.100.100"},
	}
	err = db.InsertApi(ctx, api)
	require.NoError(t, err)

	key := uid.New(16, "test")
	err = db.CreateKey(ctx, entities.Key{
		Id:          uid.Key(),
		KeyAuthId:   keyAuth.Id,
		WorkspaceId: api.WorkspaceId,
		Hash:        hash.Sha256(key),
		CreatedAt:   time.Now(),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		KeyCache: cache.NewNoopCache[entities.Key](),
		ApiCache: cache.NewNoopCache[entities.Api](),
		Database: db,
		Tracer:   tracing.NewNoop(),
	})

	buf := bytes.NewBufferString(fmt.Sprintf(`{
		"key":"%s"
		}`, key))

	req := httptest.NewRequest("POST", "/v1/keys/verify", buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Fly-Client-IP", "1.2.3.4")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)

	verifyKeyResponse := VerifyKeyResponse{}
	err = json.Unmarshal(body, &verifyKeyResponse)
	require.NoError(t, err)

	require.Equal(t, errors.FORBIDDEN, verifyKeyResponse.Code)

}

func TestVerifyKey_WithRemaining(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	db, err := database.New(database.Config{
		Logger:    logging.NewNoopLogger(),
		PrimaryUs: os.Getenv("DATABASE_DSN"),
	})
	require.NoError(t, err)

	key := uid.New(16, "test")
	remaining := int32(10)
	err = db.CreateKey(ctx, entities.Key{
		Id:          uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(key),
		CreatedAt:   time.Now(),
		Remaining:   &remaining,
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:    logging.NewNoopLogger(),
		KeyCache:  cache.NewNoopCache[entities.Key](),
		ApiCache:  cache.NewNoopCache[entities.Api](),
		Database:  db,
		Tracer:    tracing.NewNoop(),
		Ratelimit: ratelimit.NewInMemory(),
	})

	buf := bytes.NewBufferString(fmt.Sprintf(`{
		"key":"%s"
		}`, key))

	req := httptest.NewRequest("POST", "/v1/keys/verify", buf)
	req.Header.Set("Content-Type", "application/json")

	// Use up 10 requests
	for i := 9; i >= 0; i-- {

		res, err := srv.app.Test(req)
		require.NoError(t, err)
		defer res.Body.Close()

		body1, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, 200, res.StatusCode)

		vr := VerifyKeyResponse{}
		err = json.Unmarshal(body1, &vr)
		require.NoError(t, err)

		require.True(t, vr.Valid)
		require.NotNil(t, vr.Remaining)
		require.Equal(t, int32(i), *vr.Remaining)
	}

	// now it should be all used up and no longer valid

	res2, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res2.Body.Close()

	body2, err := io.ReadAll(res2.Body)
	require.NoError(t, err)
	require.Equal(t, 200, res2.StatusCode)

	verifyRes2 := VerifyKeyResponse{}
	err = json.Unmarshal(body2, &verifyRes2)
	require.NoError(t, err)

	require.False(t, verifyRes2.Valid)
	require.Equal(t, int32(0), *verifyRes2.Remaining)

}
