package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	keysv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/keys/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
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

	rootKey := &keysv1.Key{
		Id:             uid.Key(),
		KeyAuthId:      resources.UserKeyAuth.Id,
		WorkspaceId:    resources.UnkeyWorkspace.Id,
		ForWorkspaceId: util.Pointer(resources.UserWorkspace.Id),
		Hash:           hash.Sha256(uid.New(16, "test")),
		CreatedAt:      time.Now().UnixMilli(),
	}
	err := resources.Database.InsertKey(ctx, rootKey)
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		KeyCache: cache.NewNoopCache[*keysv1.Key](),
		ApiCache: cache.NewNoopCache[entities.Api](),
		Database: resources.Database,
		Tracer:   tracing.NewNoop(),
	})

	testutil.Json(t, srv.app, testutil.JsonRequest{
		Debug:      true,
		Method:     "POST",
		Path:       "/v1/internal.removeRootKey",
		Body:       fmt.Sprintf(`{"keyId": "%s"}`, rootKey.Id),
		Bearer:     resources.UserRootKey,
		StatusCode: 200,
	})

	_, found, err := resources.Database.FindKeyById(ctx, rootKey.Id)
	require.NoError(t, err)
	require.False(t, found)
}
