package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	keysv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/keys/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"
	"github.com/unkeyed/unkey/apps/agent/pkg/hash"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/keys"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

func TestV1RemoveKey(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	key := &keysv1.Key{
		Id:          uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Hash:        hash.Sha256(uid.New(16, "test")),
		CreatedAt:   time.Now().UnixMilli(),
	}
	err := resources.Database.InsertKey(ctx, key)
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		KeyCache: cache.NewNoopCache[*keysv1.Key](),
		ApiCache: cache.NewNoopCache[entities.Api](),
		Database: resources.Database,
		Tracer:   tracing.NewNoop(),
		KeyService: keys.New(keys.Config{
			Database: resources.Database,
			Events:   events.NewNoop(),
			KeyCache: cache.NewNoopCache[*keysv1.Key](),
		}),
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

	key, found, err := resources.Database.FindKeyById(ctx, key.Id)
	require.NoError(t, err)
	require.True(t, found)
	require.NotNil(t, key.DeletedAt)
}
