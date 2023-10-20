package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	keysv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/keys/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/analytics"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/keys"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/hash"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

func TestVerifyKey_Simple(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	key := uid.New(16, "test")
	err := resources.Database.InsertKey(ctx, &keysv1.Key{
		Id:          uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(key),
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		KeyCache: cache.NewNoopCache[*keysv1.Key](),
		ApiCache: cache.NewNoopCache[entities.Api](),
		Database: resources.Database,
		Tracer:   tracing.NewNoop(),
		KeyService: keys.New(keys.Config{
			Database: resources.Database,
			Events:   events.NewNoop(),
		}),
	})

	buf := bytes.NewBufferString(fmt.Sprintf(`{
		"key":"%s"
		}`, key))

	req := httptest.NewRequest("POST", "/v1/keys.verifyKey", buf)
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, 200, res.StatusCode)

	successResponse := VerifyKeyResponseV1{}
	err = json.Unmarshal(body, &successResponse)
	require.NoError(t, err)

	require.True(t, successResponse.Valid)

}

func TestVerifyKey_ReturnErrorForBadRequest(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	key := uid.New(16, "test")
	err := resources.Database.InsertKey(ctx, &keysv1.Key{
		Id:          uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(key),
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		KeyCache: cache.NewNoopCache[*keysv1.Key](),
		ApiCache: cache.NewNoopCache[entities.Api](),
		Database: resources.Database,
		Tracer:   tracing.NewNoop(),
		KeyService: keys.New(keys.Config{
			Database: resources.Database,
			Events:   events.NewNoop(),
		}),
	})

	buf := bytes.NewBufferString(fmt.Sprintf(`{
		"somethingelse":"%s"
		}`, key))

	req := httptest.NewRequest("POST", "/v1/keys.verifyKey", buf)
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, 400, res.StatusCode)

	errorResponse := errors.ErrorResponse{}
	err = json.Unmarshal(body, &errorResponse)
	require.NoError(t, err)

	require.Equal(t, errors.BAD_REQUEST, errorResponse.Error.Code)

}

func TestVerifyKey_WithTemporaryKey(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	key := uid.New(16, "test")
	err := resources.Database.InsertKey(ctx, &keysv1.Key{
		Id:          uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(key),
		CreatedAt:   time.Now().UnixMilli(),
		Expires:     util.Pointer(time.Now().Add(time.Second * 5).UnixMilli()),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		KeyCache: cache.NewNoopCache[*keysv1.Key](),
		ApiCache: cache.NewNoopCache[entities.Api](),
		Database: resources.Database,
		Tracer:   tracing.NewNoop(),
		KeyService: keys.New(keys.Config{
			Database: resources.Database,
			Events:   events.NewNoop(),
		}),
	})

	buf := bytes.NewBufferString(fmt.Sprintf(`{
		"key":"%s"
		}`, key))

	req := httptest.NewRequest("POST", "/v1/keys.verifyKey", buf)
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)

	successResponse := VerifyKeyResponseV1{}
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

	verifyKeyResponse := VerifyKeyResponseV1{}
	err = json.Unmarshal(errorBody, &verifyKeyResponse)
	require.NoError(t, err)

	require.False(t, verifyKeyResponse.Valid)

}

func TestVerifyKey_WithRatelimit(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	key := uid.New(16, "test")
	err := resources.Database.InsertKey(ctx, &keysv1.Key{
		Id:          uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(key),
		CreatedAt:   time.Now().UnixMilli(),
		Ratelimit: &keysv1.Ratelimit{
			Type:           keysv1.RatelimitType_RATELIMIT_TYPE_FAST,
			Limit:          2,
			RefillRate:     1,
			RefillInterval: 10000,
		},
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:    logging.NewNoopLogger(),
		KeyCache:  cache.NewNoopCache[*keysv1.Key](),
		ApiCache:  cache.NewNoopCache[entities.Api](),
		Database:  resources.Database,
		Tracer:    tracing.NewNoop(),
		Ratelimit: ratelimit.NewInMemory(),
		KeyService: keys.New(keys.Config{
			Database: resources.Database,
			Events:   events.NewNoop(),
		}),
	})

	buf := bytes.NewBufferString(fmt.Sprintf(`{
		"key":"%s"
		}`, key))

	req := httptest.NewRequest("POST", "/v1/keys.verifyKey", buf)
	req.Header.Set("Content-Type", "application/json")

	// first request

	res1, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res1.Body.Close()

	body1, err := io.ReadAll(res1.Body)
	require.NoError(t, err)
	require.Equal(t, 200, res1.StatusCode)

	verifyRes1 := VerifyKeyResponseV1{}
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

	verifyRes2 := VerifyKeyResponseV1{}
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

	verifyRes3 := VerifyKeyResponseV1{}
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

	verifyRes4 := VerifyKeyResponseV1{}
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

	keyAuth := entities.KeyAuth{
		Id:          uid.KeyAuth(),
		WorkspaceId: resources.UserWorkspace.Id,
	}
	err := resources.Database.InsertKeyAuth(ctx, keyAuth)
	require.NoError(t, err)

	api := entities.Api{
		Id:          uid.Api(),
		Name:        "test",
		WorkspaceId: resources.UserWorkspace.Id,
		IpWhitelist: []string{"100.100.100.100"},
		AuthType:    entities.AuthTypeKey,
		KeyAuthId:   keyAuth.Id,
	}
	err = resources.Database.InsertApi(ctx, api)
	require.NoError(t, err)

	key := uid.New(16, "test")
	err = resources.Database.InsertKey(ctx, &keysv1.Key{
		Id:          uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.Id,
		WorkspaceId: api.WorkspaceId,
		Hash:        hash.Sha256(key),
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		KeyCache: cache.NewNoopCache[*keysv1.Key](),
		ApiCache: cache.NewNoopCache[entities.Api](),
		Database: resources.Database,
		Tracer:   tracing.NewNoop(),
		KeyService: keys.New(keys.Config{
			Database: resources.Database,
			Events:   events.NewNoop(),
		}),
	})

	buf := bytes.NewBufferString(fmt.Sprintf(`{
		"key":"%s"
		}`, key))

	req := httptest.NewRequest("POST", "/v1/keys.verifyKey", buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Fly-Client-IP", "100.100.100.100")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)

	successResponse := VerifyKeyResponseV1{}
	err = json.Unmarshal(body, &successResponse)
	require.NoError(t, err)

	require.True(t, successResponse.Valid)

}

func TestVerifyKey_WithIpWhitelist_Blocked(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	keyAuth := entities.KeyAuth{
		Id:          uid.KeyAuth(),
		WorkspaceId: resources.UserWorkspace.Id,
	}
	err := resources.Database.InsertKeyAuth(ctx, keyAuth)
	require.NoError(t, err)

	api := entities.Api{
		Id:          uid.Api(),
		KeyAuthId:   keyAuth.Id,
		AuthType:    entities.AuthTypeKey,
		Name:        "test",
		WorkspaceId: resources.UserWorkspace.Id,
		IpWhitelist: []string{"100.100.100.100"},
	}
	err = resources.Database.InsertApi(ctx, api)
	require.NoError(t, err)

	key := uid.New(16, "test")
	err = resources.Database.InsertKey(ctx, &keysv1.Key{
		Id:          uid.Key(),
		KeyAuthId:   keyAuth.Id,
		WorkspaceId: api.WorkspaceId,
		Hash:        hash.Sha256(key),
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		KeyCache: cache.NewNoopCache[*keysv1.Key](),
		ApiCache: cache.NewNoopCache[entities.Api](),
		Database: resources.Database,
		Tracer:   tracing.NewNoop(),
		KeyService: keys.New(keys.Config{
			Database: resources.Database,
			Events:   events.NewNoop(),
		}),
	})

	buf := bytes.NewBufferString(fmt.Sprintf(`{
		"key":"%s"
		}`, key))

	req := httptest.NewRequest("POST", "/v1/keys.verifyKey", buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Fly-Client-IP", "1.2.3.4")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)

	verifyKeyResponse := VerifyKeyResponseV1{}
	err = json.Unmarshal(body, &verifyKeyResponse)
	require.NoError(t, err)

	require.Equal(t, errors.FORBIDDEN, verifyKeyResponse.Code)

}

func TestVerifyKey_WithRemaining(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	key := uid.New(16, "test")
	remaining := int32(10)
	err := resources.Database.InsertKey(ctx, &keysv1.Key{
		Id:          uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(key),
		CreatedAt:   time.Now().UnixMilli(),
		Remaining:   &remaining,
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:    logging.NewNoopLogger(),
		KeyCache:  cache.NewNoopCache[*keysv1.Key](),
		ApiCache:  cache.NewNoopCache[entities.Api](),
		Database:  resources.Database,
		Tracer:    tracing.NewNoop(),
		Ratelimit: ratelimit.NewInMemory(),
		Metrics:   metrics.NewNoop(),
	})

	buf := bytes.NewBufferString(fmt.Sprintf(`{
		"key":"%s"
		}`, key))

	req := httptest.NewRequest("POST", "/v1/keys.verifyKey", buf)
	req.Header.Set("Content-Type", "application/json")

	// Use up 10 requests
	for i := 9; i >= 0; i-- {

		res, err := srv.app.Test(req)
		require.NoError(t, err)
		defer res.Body.Close()

		body1, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, 200, res.StatusCode)

		vr := VerifyKeyResponseV1{}
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

	verifyRes2 := VerifyKeyResponseV1{}
	err = json.Unmarshal(body2, &verifyRes2)
	require.NoError(t, err)

	require.False(t, verifyRes2.Valid)
	require.Equal(t, int32(0), *verifyRes2.Remaining)

}

type mockAnalytics struct {
	calledPublish atomic.Int32
}

func (m *mockAnalytics) PublishKeyVerificationEvent(ctx context.Context, event analytics.KeyVerificationEvent) {
	m.calledPublish.Add(1)
}
func (m *mockAnalytics) GetKeyStats(ctx context.Context, keyId string) (analytics.KeyStats, error) {
	return analytics.KeyStats{}, fmt.Errorf("Implement me")
}

func TestVerifyKey_ShouldReportUsageWhenUsageExceeded(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	key := uid.New(16, "test")
	err := resources.Database.InsertKey(ctx, &keysv1.Key{
		Id:          uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(key),
		CreatedAt:   time.Now().UnixMilli(),
		Remaining:   util.Pointer(int32(0)),
	})
	require.NoError(t, err)

	a := &mockAnalytics{}
	srv := New(Config{
		Logger:    logging.NewNoopLogger(),
		KeyCache:  cache.NewNoopCache[*keysv1.Key](),
		ApiCache:  cache.NewNoopCache[entities.Api](),
		Database:  resources.Database,
		Tracer:    tracing.NewNoop(),
		Analytics: a,
		KeyService: keys.New(keys.Config{
			Database: resources.Database,
			Events:   events.NewNoop(),
		}),
	})

	buf := bytes.NewBufferString(fmt.Sprintf(`{
		"key":"%s"
		}`, key))

	req := httptest.NewRequest("POST", "/v1/keys.verifyKey", buf)
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, 200, res.StatusCode)

	successResponse := VerifyKeyResponseV1{}
	err = json.Unmarshal(body, &successResponse)
	require.NoError(t, err)

	require.False(t, successResponse.Valid)
	require.Equal(t, int32(1), a.calledPublish.Load())

}
