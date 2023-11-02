package server_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/analytics"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/server"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/apis"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/keys"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"

	"github.com/stretchr/testify/require"
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
	err := resources.Database.InsertKey(ctx, &authenticationv1.Key{
		KeyId:       uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.KeyAuthId,
		WorkspaceId: resources.UserWorkspace.WorkspaceId,
		Hash:        hash.Sha256(key),
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	srv := testutil.NewServer(t, resources)
	res := testutil.Json[server.VerifyKeyResponseV1](t, srv.App, testutil.JsonRequest{

		Path:       "/v1/keys.verifyKey",
		Body:       fmt.Sprintf(`{"key":"%s"}`, key),
		StatusCode: 200,
		Bearer:     key,
	})

	require.True(t, res.Valid)

}

func TestVerifyKey_ReturnErrorForBadRequest(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	key := uid.New(16, "test")
	err := resources.Database.InsertKey(ctx, &authenticationv1.Key{
		KeyId:       uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.KeyAuthId,
		WorkspaceId: resources.UserWorkspace.WorkspaceId,
		Hash:        hash.Sha256(key),
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	srv := testutil.NewServer(t, resources)
	res := testutil.Json[server.ErrorResponse](t, srv.App, testutil.JsonRequest{

		Path:       "/v1/keys.verifyKey",
		Body:       fmt.Sprintf(`{"somethingelse":"%s"}`, key),
		StatusCode: 400,
		Bearer:     key,
	})

	require.Equal(t, "BAD_REQUEST", res.Error.Code)

}

func TestVerifyKey_WithTemporaryKey(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	key := uid.New(16, "test")
	err := resources.Database.InsertKey(ctx, &authenticationv1.Key{
		KeyId:       uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.KeyAuthId,
		WorkspaceId: resources.UserWorkspace.WorkspaceId,
		Hash:        hash.Sha256(key),
		CreatedAt:   time.Now().UnixMilli(),
		Expires:     util.Pointer(time.Now().Add(time.Second * 5).UnixMilli()),
	})
	require.NoError(t, err)

	srv := testutil.NewServer(t, resources)
	successRes := testutil.Json[server.VerifyKeyResponseV1](t, srv.App, testutil.JsonRequest{

		Path:       "/v1/keys.verifyKey",
		Body:       fmt.Sprintf(`{"key":"%s"}`, key),
		StatusCode: 200,
		Bearer:     key,
	})

	require.True(t, successRes.Valid)

	// wait until key expires
	time.Sleep(time.Second * 10)

	errorRes := testutil.Json[server.VerifyKeyResponseV1](t, srv.App, testutil.JsonRequest{

		Path:       "/v1/keys.verifyKey",
		Body:       fmt.Sprintf(`{"key":"%s"}`, key),
		StatusCode: 200,
		Bearer:     key,
	})
	require.False(t, errorRes.Valid)
	require.Equal(t, "NOT_FOUND", errorRes.Code)

}

func TestVerifyKey_WithRatelimit(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	key := uid.New(16, "test")
	err := resources.Database.InsertKey(ctx, &authenticationv1.Key{
		KeyId:       uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.KeyAuthId,
		WorkspaceId: resources.UserWorkspace.WorkspaceId,
		Hash:        hash.Sha256(key),
		CreatedAt:   time.Now().UnixMilli(),
		Ratelimit: &authenticationv1.Ratelimit{
			Type:           authenticationv1.RatelimitType_RATELIMIT_TYPE_FAST,
			Limit:          2,
			RefillRate:     1,
			RefillInterval: 10000,
		},
	})
	require.NoError(t, err)

	srv := testutil.NewServer(t, resources)
	res1 := testutil.Json[server.VerifyKeyResponseV1](t, srv.App, testutil.JsonRequest{

		Path:       "/v1/keys.verifyKey",
		Body:       fmt.Sprintf(`{"key":"%s"}`, key),
		StatusCode: 200,
		Bearer:     key,
	})

	require.True(t, res1.Valid)
	require.Equal(t, int32(2), res1.Ratelimit.Limit)
	require.Equal(t, int32(1), res1.Ratelimit.Remaining)
	require.GreaterOrEqual(t, res1.Ratelimit.Reset, int64(time.Now().UnixMilli()))
	require.LessOrEqual(t, res1.Ratelimit.Reset, int64(time.Now().Add(time.Second*10).UnixMilli()))

	// second request

	res2 := testutil.Json[server.VerifyKeyResponseV1](t, srv.App, testutil.JsonRequest{

		Path:       "/v1/keys.verifyKey",
		Body:       fmt.Sprintf(`{"key":"%s"}`, key),
		StatusCode: 200,
		Bearer:     key,
	})

	require.True(t, res2.Valid)
	require.Equal(t, int32(2), res2.Ratelimit.Limit)
	require.Equal(t, int32(0), res2.Ratelimit.Remaining)
	require.GreaterOrEqual(t, res2.Ratelimit.Reset, int64(time.Now().UnixMilli()))
	require.LessOrEqual(t, res2.Ratelimit.Reset, int64(time.Now().Add(time.Second*10).UnixMilli()))

	// third request

	res3 := testutil.Json[server.VerifyKeyResponseV1](t, srv.App, testutil.JsonRequest{

		Path:       "/v1/keys.verifyKey",
		Body:       fmt.Sprintf(`{"key":"%s"}`, key),
		StatusCode: 200,
		Bearer:     key,
	})
	require.False(t, res3.Valid)
	require.Equal(t, int32(2), res3.Ratelimit.Limit)
	require.Equal(t, int32(0), res3.Ratelimit.Remaining)
	require.GreaterOrEqual(t, res3.Ratelimit.Reset, int64(time.Now().UnixMilli()))
	require.LessOrEqual(t, res3.Ratelimit.Reset, int64(time.Now().Add(time.Second*10).UnixMilli()))

	// wait and try again in the next window
	time.Sleep(time.Until(time.UnixMilli(res3.Ratelimit.Reset)))
	res4 := testutil.Json[server.VerifyKeyResponseV1](t, srv.App, testutil.JsonRequest{

		Path:       "/v1/keys.verifyKey",
		Body:       fmt.Sprintf(`{"key":"%s"}`, key),
		StatusCode: 200,
		Bearer:     key,
	})
	require.True(t, res4.Valid)
	require.Equal(t, int32(2), res4.Ratelimit.Limit)
	require.Equal(t, int32(0), res4.Ratelimit.Remaining)
	require.GreaterOrEqual(t, res4.Ratelimit.Reset, int64(time.Now().UnixMilli()))
	require.LessOrEqual(t, res4.Ratelimit.Reset, int64(time.Now().Add(time.Second*10).UnixMilli()))

}

func TestVerifyKey_WithIpWhitelist_Pass(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	keyAuth := &authenticationv1.KeyAuth{
		KeyAuthId:   uid.KeyAuth(),
		WorkspaceId: resources.UserWorkspace.WorkspaceId,
	}
	err := resources.Database.InsertKeyAuth(ctx, keyAuth)
	require.NoError(t, err)

	api := &apisv1.Api{
		ApiId:       uid.Api(),
		Name:        "test",
		WorkspaceId: resources.UserWorkspace.WorkspaceId,
		IpWhitelist: []string{"100.100.100.100"},
		AuthType:    apisv1.AuthType_AUTH_TYPE_KEY,
		KeyAuthId:   &keyAuth.KeyAuthId,
	}
	err = resources.Database.InsertApi(ctx, api)
	require.NoError(t, err)

	key := uid.New(16, "test")
	err = resources.Database.InsertKey(ctx, &authenticationv1.Key{
		KeyId:       uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.KeyAuthId,
		WorkspaceId: api.WorkspaceId,
		Hash:        hash.Sha256(key),
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	srv := testutil.NewServer(t, resources)

	res := testutil.Json[server.VerifyKeyResponseV1](t, srv.App, testutil.JsonRequest{

		Path:       "/v1/keys.verifyKey",
		Body:       fmt.Sprintf(`{"key":"%s"}`, key),
		StatusCode: 200,
		Bearer:     key,
		RequestHeaders: map[string]string{
			"Fly-Client-IP": "100.100.100.100",
		},
	})
	require.True(t, res.Valid)

}

func TestVerifyKey_WithIpWhitelist_Blocked(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	keyAuth := &authenticationv1.KeyAuth{
		KeyAuthId:   uid.KeyAuth(),
		WorkspaceId: resources.UserWorkspace.WorkspaceId,
	}
	err := resources.Database.InsertKeyAuth(ctx, keyAuth)
	require.NoError(t, err)

	api := &apisv1.Api{
		ApiId:       uid.Api(),
		KeyAuthId:   &keyAuth.KeyAuthId,
		AuthType:    apisv1.AuthType_AUTH_TYPE_KEY,
		Name:        "test",
		WorkspaceId: resources.UserWorkspace.WorkspaceId,
		IpWhitelist: []string{"100.100.100.100"},
	}
	err = resources.Database.InsertApi(ctx, api)
	require.NoError(t, err)

	key := uid.New(16, "test")
	err = resources.Database.InsertKey(ctx, &authenticationv1.Key{
		KeyId:       uid.Key(),
		KeyAuthId:   keyAuth.KeyAuthId,
		WorkspaceId: api.WorkspaceId,
		Hash:        hash.Sha256(key),
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	srv := testutil.NewServer(t, resources)

	res := testutil.Json[server.VerifyKeyResponseV1](t, srv.App, testutil.JsonRequest{

		Path:       "/v1/keys.verifyKey",
		Body:       fmt.Sprintf(`{"key":"%s"}`, key),
		StatusCode: 200,
		RequestHeaders: map[string]string{
			"Fly-Client-IP": "1.2.3.4",
		},
		Bearer: key,
	})

	require.Equal(t, "FORBIDDEN", res.Code)

}

func TestVerifyKey_WithRemaining(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	key := uid.New(16, "test")
	remaining := int32(10)
	err := resources.Database.InsertKey(ctx, &authenticationv1.Key{
		KeyId:       uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.KeyAuthId,
		WorkspaceId: resources.UserWorkspace.WorkspaceId,
		Hash:        hash.Sha256(key),
		CreatedAt:   time.Now().UnixMilli(),
		Remaining:   &remaining,
	})
	require.NoError(t, err)

	srv := testutil.NewServer(t, resources)

	// Use up 10 requests
	for i := 9; i >= 0; i-- {

		res := testutil.Json[server.VerifyKeyResponseV1](t, srv.App, testutil.JsonRequest{

			Path:       "/v1/keys.verifyKey",
			Body:       fmt.Sprintf(`{"key":"%s"}`, key),
			StatusCode: 200,
			Bearer:     key,
		})
		require.True(t, res.Valid)
		require.NotNil(t, res.Remaining)
		require.Equal(t, int32(i), *res.Remaining)
	}

	// now it should be all used up and no longer valid

	res2 := testutil.Json[server.VerifyKeyResponseV1](t, srv.App, testutil.JsonRequest{

		Path:       "/v1/keys.verifyKey",
		Body:       fmt.Sprintf(`{"key":"%s"}`, key),
		StatusCode: 200,
		Bearer:     key,
	})
	require.False(t, res2.Valid)
	require.Equal(t, int32(0), *res2.Remaining)

}

type mockAnalytics struct {
	calledPublish atomic.Int32
}

func (m *mockAnalytics) PublishKeyVerificationEvent(ctx context.Context, event analytics.KeyVerificationEvent) {
	m.calledPublish.Add(1)
}
func (m *mockAnalytics) GetKeyStats(ctx context.Context, workspaceId, apiId, keyId string) (analytics.KeyStats, error) {
	return analytics.KeyStats{}, fmt.Errorf("Implement me")
}

func TestVerifyKey_ShouldReportUsageWhenUsageExceeded(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	key := uid.New(16, "test")
	err := resources.Database.InsertKey(ctx, &authenticationv1.Key{
		KeyId:       uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.KeyAuthId,
		WorkspaceId: resources.UserWorkspace.WorkspaceId,
		Hash:        hash.Sha256(key),
		CreatedAt:   time.Now().UnixMilli(),
		Remaining:   util.Pointer(int32(0)),
	})
	require.NoError(t, err)

	a := &mockAnalytics{}
	srv := server.New(server.Config{
		Logger:    logging.NewNoop(),
		KeyCache:  cache.NewNoopCache[*authenticationv1.Key](),
		ApiCache:  cache.NewNoopCache[*apisv1.Api](),
		Database:  resources.Database,
		Tracer:    tracing.NewNoop(),
		Analytics: a,
		KeyService: keys.New(keys.Config{
			Database:           resources.Database,
			Events:             events.NewNoop(),
			Logger:             logging.NewNoop(),
			KeyCache:           cache.NewNoopCache[*authenticationv1.Key](),
			ApiCache:           cache.NewNoopCache[*apisv1.Api](),
			Tracer:             tracing.NewNoop(),
			Metrics:            metrics.NewNoop(),
			Analytics:          a,
			MemoryRatelimit:    ratelimit.NewInMemory(),
			ConsitentRatelimit: ratelimit.NewInMemory(),
		}),
		ApiService: apis.New(apis.Config{
			Database: resources.Database,
		}),
	})

	res := testutil.Json[server.VerifyKeyResponseV1](t, srv.App, testutil.JsonRequest{

		Path:       "/v1/keys.verifyKey",
		Body:       fmt.Sprintf(`{"key":"%s"}`, key),
		StatusCode: 200,
		Bearer:     key,
	})

	require.False(t, res.Valid)
	require.Equal(t, int32(1), a.calledPublish.Load())

}
