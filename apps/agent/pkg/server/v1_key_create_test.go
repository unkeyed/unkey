package server_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/server"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
)

func TestCreateKey_Simple(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := testutil.NewServer(t, resources)

	res := testutil.Json[server.CreateKeyResponse](t, srv.App, testutil.JsonRequest{

		Path:       "/v1/keys",
		Bearer:     resources.UserRootKey,
		Body:       fmt.Sprintf(`{"apiId":"%s"}`, resources.UserApi.ApiId),
		StatusCode: 200,
	})

	require.NotEmpty(t, res.Key)
	require.NotEmpty(t, res.KeyId)

	foundKey, found, err := resources.Database.FindKeyById(ctx, res.KeyId)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, res.KeyId, foundKey.KeyId)
}

func TestCreateKey_RejectInvalidRatelimitTypes(t *testing.T) {
	t.Parallel()

	resources := testutil.SetupResources(t)

	srv := testutil.NewServer(t, resources)

	res := testutil.Json[server.ErrorResponse](t, srv.App, testutil.JsonRequest{

		Path:       "/v1/keys",
		Bearer:     resources.UserRootKey,
		Body:       fmt.Sprintf(`{"apiId":"%s", "ratelimit": {"type": "x"}}`, resources.UserApi.ApiId),
		StatusCode: 400,
	})

	require.Equal(t, "BAD_REQUEST", res.Error.Code)

}

func TestCreateKey_StartIncludesPrefix(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := testutil.NewServer(t, resources)
	res := testutil.Json[server.CreateKeyResponse](t, srv.App, testutil.JsonRequest{

		Path:       "/v1/keys",
		Bearer:     resources.UserRootKey,
		Body:       fmt.Sprintf(`{"apiId":"%s", "byteLength": 32, "prefix": "test"}`, resources.UserApi.ApiId),
		StatusCode: 200,
	})

	require.NotEmpty(t, res.Key)
	require.NotEmpty(t, res.KeyId)

	foundKey, found, err := resources.Database.FindKeyById(ctx, res.KeyId)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, res.KeyId, foundKey.KeyId)
	require.True(t, strings.HasPrefix(foundKey.Start, "test_"))

}

func TestCreateKey_WithCustom(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := testutil.NewServer(t, resources)

	res := testutil.Json[server.CreateKeyResponse](t, srv.App, testutil.JsonRequest{
		Method: "POST",
		Path:   "/v1/keys",
		Bearer: resources.UserRootKey,
		Body: fmt.Sprintf(`{
			"apiId":"%s",
			"byteLength": 32,
			"prefix": "test",
			"ownerId": "chronark",
			"expires": %d,
			"ratelimit":{
				"type":"fast",
				"limit": 10,
				"refillRate":10,
				"refillInterval":1000
			}
			}`, resources.UserApi.ApiId, time.Now().Add(time.Hour).UnixMilli()),
		StatusCode: 200,
	})

	require.NotEmpty(t, res.Key)
	require.NotEmpty(t, res.KeyId)

	foundKey, found, err := resources.Database.FindKeyById(ctx, res.KeyId)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, res.KeyId, foundKey.KeyId)
	require.GreaterOrEqual(t, len(res.Key), 30)
	require.True(t, strings.HasPrefix(res.Key, "test_"))
	require.Equal(t, "chronark", *foundKey.OwnerId)
	require.True(t, time.UnixMilli(foundKey.GetExpires()).After(time.Now()))

	require.NotNil(t, foundKey.Ratelimit)
	require.Equal(t, authenticationv1.RatelimitType_RATELIMIT_TYPE_FAST, foundKey.Ratelimit.Type)
	require.Equal(t, int32(10), foundKey.Ratelimit.Limit)
	require.Equal(t, int32(10), foundKey.Ratelimit.RefillRate)
	require.Equal(t, int32(1000), foundKey.Ratelimit.RefillInterval)

}

func TestCreateKey_WithRemanining(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := testutil.NewServer(t, resources)

	res := testutil.Json[server.CreateKeyResponse](t, srv.App, testutil.JsonRequest{
		Method: "POST",
		Path:   "/v1/keys",
		Bearer: resources.UserRootKey,
		Body: fmt.Sprintf(`{
			"apiId":"%s",
			"remaining": 4
		}`, resources.UserApi.ApiId),
		StatusCode: 200,
	})

	require.NotEmpty(t, res.Key)
	require.NotEmpty(t, res.KeyId)

	foundKey, found, err := resources.Database.FindKeyById(ctx, res.KeyId)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, res.KeyId, foundKey.KeyId)
	require.NotNil(t, foundKey.Remaining)
	require.Equal(t, int32(4), *foundKey.Remaining)

}
