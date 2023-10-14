package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/hash"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

func TestV1RemoveKey(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	key := entities.Key{
		Id:          uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(uid.New(16, "test")),
		CreatedAt:   time.Now(),
	}
	err := resources.Database.InsertKey(ctx, key)
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		KeyCache: cache.NewNoopCache[entities.Key](),
		ApiCache: cache.NewNoopCache[entities.Api](),
		Database: resources.Database,
		Tracer:   tracing.NewNoop(),
	})

	res := RemoveKeyResponseV1{}
	testutil.Json(t, srv.app, testutil.JsonRequest{
		Method:     "POST",
		Path:       "/v1/keys.removeKey",
		Body:       fmt.Sprintf(`{"keyId": "%s"}`, key.Id),
		Bearer:     resources.UserRootKey,
		Response:   &res,
		StatusCode: 200,
	})

	_, found, err := resources.Database.FindKeyById(ctx, key.Id)
	require.NoError(t, err)
	require.False(t, found)
}
