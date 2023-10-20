package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	keysv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/keys/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/keys"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

func TestCreateKey_Simple(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

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
		"apiId":"%s"
		}`, resources.UserApi.Id))

	req := httptest.NewRequest("POST", "/v1/keys", buf)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UserRootKey))
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, res.StatusCode, 200)

	createKeyResponse := CreateKeyResponse{}
	err = json.Unmarshal(body, &createKeyResponse)
	require.NoError(t, err)

	require.NotEmpty(t, createKeyResponse.Key)
	require.NotEmpty(t, createKeyResponse.KeyId)

	foundKey, found, err := resources.Database.FindKeyById(ctx, createKeyResponse.KeyId)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, createKeyResponse.KeyId, foundKey.Id)
}

func TestCreateKey_RejectInvalidRatelimitTypes(t *testing.T) {
	t.Parallel()

	resources := testutil.SetupResources(t)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		KeyCache: cache.NewNoopCache[*keysv1.Key](),
		ApiCache: cache.NewNoopCache[entities.Api](),
		Database: resources.Database,
		Tracer:   tracing.NewNoop(),
	})

	buf := bytes.NewBufferString(fmt.Sprintf(`{
		"apiId":"%s",
		"ratelimit": {
			"type": "x"
			}
		}`, resources.UserApi.Id))

	req := httptest.NewRequest("POST", "/v1/keys", buf)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UserRootKey))
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, res.StatusCode, 400)

	createKeyResponse := errors.ErrorResponse{}
	err = json.Unmarshal(body, &createKeyResponse)
	require.NoError(t, err)

	require.Equal(t, "BAD_REQUEST", createKeyResponse.Error.Code)

}

func TestCreateKey_StartIncludesPrefix(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

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
		"apiId":"%s",
		"byteLength": 32,
		"prefix": "test"
		}`, resources.UserApi.Id))

	req := httptest.NewRequest("POST", "/v1/keys", buf)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UserRootKey))
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, 200, res.StatusCode)

	createKeyResponse := CreateKeyResponse{}
	err = json.Unmarshal(body, &createKeyResponse)
	require.NoError(t, err)

	require.NotEmpty(t, createKeyResponse.Key)
	require.NotEmpty(t, createKeyResponse.KeyId)

	foundKey, found, err := resources.Database.FindKeyById(ctx, createKeyResponse.KeyId)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, createKeyResponse.KeyId, foundKey.Id)
	require.True(t, strings.HasPrefix(foundKey.Start, "test_"))

}

func TestCreateKey_WithCustom(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

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
		}`, resources.UserApi.Id, time.Now().Add(time.Hour).UnixMilli()))

	req := httptest.NewRequest("POST", "/v1/keys", buf)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UserRootKey))
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, 200, res.StatusCode)

	createKeyResponse := CreateKeyResponse{}
	err = json.Unmarshal(body, &createKeyResponse)
	require.NoError(t, err)

	require.NotEmpty(t, createKeyResponse.Key)
	require.NotEmpty(t, createKeyResponse.KeyId)

	foundKey, found, err := resources.Database.FindKeyById(ctx, createKeyResponse.KeyId)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, createKeyResponse.KeyId, foundKey.Id)
	require.GreaterOrEqual(t, len(createKeyResponse.Key), 30)
	require.True(t, strings.HasPrefix(createKeyResponse.Key, "test_"))
	require.Equal(t, "chronark", *foundKey.OwnerId)
	require.True(t, time.UnixMilli(foundKey.GetExpires()).After(time.Now()))

	require.NotNil(t, foundKey.Ratelimit)
	require.Equal(t, keysv1.RatelimitType_RATELIMIT_TYPE_FAST, foundKey.Ratelimit.Type)
	require.Equal(t, int32(10), foundKey.Ratelimit.Limit)
	require.Equal(t, int32(10), foundKey.Ratelimit.RefillRate)
	require.Equal(t, int32(1000), foundKey.Ratelimit.RefillInterval)

}

func TestCreateKey_WithRemanining(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

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
		"apiId":"%s",
		"remaining":4
		}`, resources.UserApi.Id))

	req := httptest.NewRequest("POST", "/v1/keys", buf)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UserRootKey))
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, 200, res.StatusCode)

	createKeyResponse := CreateKeyResponse{}
	err = json.Unmarshal(body, &createKeyResponse)
	require.NoError(t, err)

	require.NotEmpty(t, createKeyResponse.Key)
	require.NotEmpty(t, createKeyResponse.KeyId)

	foundKey, found, err := resources.Database.FindKeyById(ctx, createKeyResponse.KeyId)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, createKeyResponse.KeyId, foundKey.Id)
	require.NotNil(t, foundKey.Remaining)
	require.Equal(t, int32(4), *foundKey.Remaining)

}
