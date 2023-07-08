package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/api/pkg/cache"
	"github.com/unkeyed/unkey/apps/api/pkg/database"
	"github.com/unkeyed/unkey/apps/api/pkg/entities"
	"github.com/unkeyed/unkey/apps/api/pkg/logging"
	"github.com/unkeyed/unkey/apps/api/pkg/testutil"
	"github.com/unkeyed/unkey/apps/api/pkg/tracing"
)

func TestCreateKey_Simple(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	db, err := database.New(database.Config{
		PrimaryUs: os.Getenv("DATABASE_DSN"),
		Logger:    logging.NewNoopLogger(),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.New(),
		Cache:    cache.NewNoopCache[entities.Key](),
		Database: db,
		Tracer:   tracing.NewNoop(),
	})

	buf := bytes.NewBufferString(fmt.Sprintf(`{
		"apiId":"%s"
		}`, resources.UserApi.Id))

	req := httptest.NewRequest("POST", "/v1/keys", buf)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UnkeyKey))
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

	found, err := db.GetKeyById(ctx, createKeyResponse.KeyId)
	require.NoError(t, err)
	require.Equal(t, createKeyResponse.KeyId, found.Id)
}

func TestCreateKey_WithCustom(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	db, err := database.New(database.Config{Logger: logging.NewNoopLogger(),

		PrimaryUs: os.Getenv("DATABASE_DSN"),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		Cache:    cache.NewNoopCache[entities.Key](),
		Database: db,
		Tracer:   tracing.NewNoop(),
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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UnkeyKey))
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

	found, err := db.GetKeyById(ctx, createKeyResponse.KeyId)
	require.NoError(t, err)
	require.Equal(t, createKeyResponse.KeyId, found.Id)
	require.GreaterOrEqual(t, len(createKeyResponse.Key), 30)
	require.True(t, strings.HasPrefix(createKeyResponse.Key, "test_"))
	require.Equal(t, "chronark", found.OwnerId)
	require.True(t, found.Expires.After(time.Now()))

	require.Equal(t, "fast", found.Ratelimit.Type)
	require.Equal(t, int64(10), found.Ratelimit.Limit)
	require.Equal(t, int64(10), found.Ratelimit.RefillRate)
	require.Equal(t, int64(1000), found.Ratelimit.RefillInterval)

}
