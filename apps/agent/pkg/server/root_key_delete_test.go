package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"

	"github.com/unkeyed/unkey/apps/agent/pkg/hash"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

func TestDeleteRootKey(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	rootKey := &authenticationv1.Key{
		KeyId:          uid.Key(),
		KeyAuthId:      resources.UserKeyAuth.KeyAuthId,
		WorkspaceId:    resources.UnkeyWorkspace.WorkspaceId,
		ForWorkspaceId: util.Pointer(resources.UserWorkspace.WorkspaceId),
		Hash:           hash.Sha256(uid.New(16, "test")),
		CreatedAt:      time.Now().UnixMilli(),
	}
	err := resources.Database.InsertKey(ctx, rootKey)
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoop(),
		KeyCache: cache.NewNoopCache[*authenticationv1.Key](),
		ApiCache: cache.NewNoopCache[*apisv1.Api](),
		Database: resources.Database,
		Tracer:   tracing.NewNoop(),
	})

	testutil.Json(t, srv.app, testutil.JsonRequest{
		Method:     "POST",
		Path:       "/v1/internal.removeRootKey",
		Body:       fmt.Sprintf(`{"keyId": "%s"}`, rootKey.KeyId),
		Bearer:     resources.UserRootKey,
		StatusCode: 200,
	})

	key, found, err := resources.Database.FindKeyById(ctx, rootKey.KeyId)
	require.NoError(t, err)
	require.True(t, found)
	require.NotNil(t, key.DeletedAt)
}
