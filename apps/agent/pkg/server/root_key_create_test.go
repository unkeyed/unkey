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

func TestRootCreateKey_Simple(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := testutil.NewServer(t, resources)

	res := testutil.Json[server.CreateRootKeyResponse](t, srv.App, testutil.JsonRequest{

		Path:       "/v1/internal/rootkeys",
		Bearer:     resources.UnkeyAppAuthToken,
		Body:       fmt.Sprintf(`{"name":"simple","forWorkspaceId":"%s"}`, resources.UnkeyWorkspace.WorkspaceId),
		StatusCode: 200,
	})

	require.NotEmpty(t, res.Key)
	require.NotEmpty(t, res.KeyId)

	foundKey, found, err := resources.Database.FindKeyById(ctx, res.KeyId)
	require.NoError(t, err)
	require.True(t, found)

	require.Equal(t, res.KeyId, foundKey.KeyId)
}

func TestRootCreateKey_WithExpiry(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := testutil.NewServer(t, resources)

	res := testutil.Json[server.CreateRootKeyResponse](t, srv.App, testutil.JsonRequest{

		Path:       "/v1/internal/rootkeys",
		Bearer:     resources.UnkeyAppAuthToken,
		Body:       fmt.Sprintf(`{"name":"simple","forWorkspaceId":"%s", "expires": %d}`, resources.UnkeyWorkspace.WorkspaceId, time.Now().Add(time.Hour).UnixMilli()),
		StatusCode: 200,
	})

	require.NotEmpty(t, res.Key)
	require.NotEmpty(t, res.KeyId)
	foundKey, found, err := resources.Database.FindKeyById(ctx, res.KeyId)
	require.True(t, found)
	require.NoError(t, err)
	require.Equal(t, res.KeyId, foundKey.KeyId)
	require.GreaterOrEqual(t, len(res.Key), 30)
	require.True(t, strings.HasPrefix(res.Key, "unkey_"))
	require.True(t, time.UnixMilli(foundKey.GetExpires()).After(time.Now()))

	require.Equal(t, authenticationv1.RatelimitType_RATELIMIT_TYPE_FAST, foundKey.Ratelimit.Type)
	require.Equal(t, int32(100), foundKey.Ratelimit.Limit)
	require.Equal(t, int32(10), foundKey.Ratelimit.RefillRate)
	require.Equal(t, int32(1000), foundKey.Ratelimit.RefillInterval)

}
