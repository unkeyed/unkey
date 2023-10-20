package server

import (
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
	"github.com/unkeyed/unkey/apps/agent/pkg/events"
	"github.com/unkeyed/unkey/apps/agent/pkg/hash"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/keys"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

func TestKeyFindV1_Simple(t *testing.T) {
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

	key := &keysv1.Key{
		Id:          uid.Key(),
		KeyAuthId:   resources.UserKeyAuth.Id,
		WorkspaceId: resources.UserWorkspace.Id,
		Name:        util.Pointer("test"),
		Hash:        hash.Sha256(uid.New(16, "test")),
		CreatedAt:   time.Now().UnixMilli(),
	}
	err := resources.Database.InsertKey(ctx, key)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/keys/%s", key.Id), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UserRootKey))

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, 200, res.StatusCode)

	successResponse := GetKeyResponse{}
	err = json.Unmarshal(body, &successResponse)
	require.NoError(t, err)

	require.Equal(t, key.Id, successResponse.Id)
	require.Equal(t, resources.UserApi.Id, successResponse.ApiId)
	require.Equal(t, key.WorkspaceId, successResponse.WorkspaceId)
	require.Equal(t, *key.Name, successResponse.Name)
	require.True(t, strings.HasPrefix(key.Hash, successResponse.Start))
	require.WithinDuration(t, time.UnixMilli(key.GetCreatedAt()), time.UnixMilli(successResponse.CreatedAt), time.Second)

}
