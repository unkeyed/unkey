package server_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/hash"
	"github.com/unkeyed/unkey/apps/agent/pkg/server"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

func TestKeyFindV1_Simple(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	resources := testutil.SetupResources(t)

	srv := testutil.NewServer(t, resources)

	key := &authenticationv1.Key{
		KeyId:       uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.KeyAuthId,
		WorkspaceId: resources.UserWorkspace.WorkspaceId,
		Name:        util.Pointer("test"),
		Hash:        hash.Sha256(uid.New(16, "test")),
		CreatedAt:   time.Now().UnixMilli(),
	}
	err := resources.Database.InsertKey(ctx, key)
	require.NoError(t, err)

	res := testutil.Json[server.GetKeyResponse](t, srv.App, testutil.JsonRequest{
		Method:     "GET",
		Path:       fmt.Sprintf("/v1/keys/%s", key.KeyId),
		Bearer:     resources.UserRootKey,
		StatusCode: 200,
	})

	require.Equal(t, key.KeyId, res.Id)
	require.Equal(t, resources.UserApi.ApiId, res.ApiId)
	require.Equal(t, key.WorkspaceId, res.WorkspaceId)
	require.Equal(t, *key.Name, res.Name)
	require.True(t, strings.HasPrefix(key.Hash, res.Start))
	require.WithinDuration(t, time.UnixMilli(key.GetCreatedAt()), time.UnixMilli(res.CreatedAt), time.Second)

}
