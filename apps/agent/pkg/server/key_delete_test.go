package server

import (
	"context"
	"fmt"
	"io"
	"net/http/httptest"
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

func TestDeleteKey(t *testing.T) {
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

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/v1/keys/%s", key.Id), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UserRootKey))

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	_, err = io.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, 200, res.StatusCode)

	_, found, err := resources.Database.FindKeyById(ctx, key.Id)
	require.NoError(t, err)
	require.False(t, found)
}
