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

	"github.com/chronark/unkey/apps/api/pkg/cache"
	"github.com/chronark/unkey/apps/api/pkg/entities"
	"github.com/chronark/unkey/apps/api/pkg/logging"
	"github.com/chronark/unkey/apps/api/pkg/testutil"
	"github.com/chronark/unkey/apps/api/pkg/tracing"
	"github.com/stretchr/testify/require"
)

func TestRootCreateKey_Simple(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := New(Config{
		Logger:            logging.New(),
		Cache:             cache.NewInMemoryCache[entities.Key](),
		Database:          resources.Database,
		Tracer:            tracing.NewNoop(),
		UnkeyWorkspaceId:  resources.UnkeyWorkspace.Id,
		UnkeyApiId:        resources.UnkeyApi.Id,
		UnkeyAppAuthToken: "supersecret",
	})

	buf := bytes.NewBufferString(fmt.Sprintf(`{
		"name":"simple",
		"forWorkspaceId":"%s"
		}`, resources.UserWorkspace.Id))

	req := httptest.NewRequest("POST", "/v1/internal/rootkeys", buf)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", srv.unkeyAppAuthToken))
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, res.StatusCode, 200)

	createRootKeyResponse := CreateRootKeyResponse{}
	err = json.Unmarshal(body, &createRootKeyResponse)
	require.NoError(t, err)

	require.NotEmpty(t, createRootKeyResponse.Key)
	require.NotEmpty(t, createRootKeyResponse.KeyId)

	found, err := resources.Database.GetKeyById(ctx, createRootKeyResponse.KeyId)
	require.NoError(t, err)
	require.Equal(t, createRootKeyResponse.KeyId, found.Id)
}

func TestRootCreateKey_WithExpiry(t *testing.T) {
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := New(Config{
		Logger:            logging.NewNoopLogger(),
		Cache:             cache.NewInMemoryCache[entities.Key](),
		Database:          resources.Database,
		Tracer:            tracing.NewNoop(),
		UnkeyAppAuthToken: "supersecret",
		UnkeyWorkspaceId:  resources.UnkeyWorkspace.Id,
		UnkeyApiId:        resources.UnkeyApi.Id,
	})

	buf := bytes.NewBufferString(fmt.Sprintf(`{
		"name":"simple",
		"forWorkspaceId":"%s",
		"expires": %d
		}`, resources.UserWorkspace.Id, time.Now().Add(time.Hour).UnixMilli()))

	req := httptest.NewRequest("POST", "/v1/internal/rootkeys", buf)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", srv.unkeyAppAuthToken))
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, 200, res.StatusCode)

	createRootKeyResponse := CreateRootKeyResponse{}
	err = json.Unmarshal(body, &createRootKeyResponse)
	require.NoError(t, err)

	require.NotEmpty(t, createRootKeyResponse.Key)
	require.NotEmpty(t, createRootKeyResponse.KeyId)

	found, err := resources.Database.GetKeyById(ctx, createRootKeyResponse.KeyId)
	require.NoError(t, err)
	require.Equal(t, createRootKeyResponse.KeyId, found.Id)
	require.GreaterOrEqual(t, len(createRootKeyResponse.Key), 30)
	require.True(t, strings.HasPrefix(createRootKeyResponse.Key, "unkey_"))
	require.True(t, found.Expires.After(time.Now()))

	require.Equal(t, "fast", found.Ratelimit.Type)
	require.Equal(t, int64(100), found.Ratelimit.Limit)
	require.Equal(t, int64(10), found.Ratelimit.RefillRate)
	require.Equal(t, int64(1000), found.Ratelimit.RefillInterval)

}
