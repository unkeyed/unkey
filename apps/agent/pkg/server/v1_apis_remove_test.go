package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	keysv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/keys/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/apis"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/keys"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/workspaces"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

func TestV1ApisRemove(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := New(Config{
		Logger:            logging.NewNoopLogger(),
		KeyCache:          cache.NewNoopCache[*keysv1.Key](),
		ApiCache:          cache.NewNoopCache[entities.Api](),
		Database:          resources.Database,
		Tracer:            tracing.NewNoop(),
		UnkeyWorkspaceId:  resources.UnkeyWorkspace.Id,
		UnkeyApiId:        resources.UnkeyApi.Id,
		UnkeyAppAuthToken: "supersecret",
		WorkspaceService:  workspaces.New(workspaces.Config{Database: resources.Database}),
		ApiService:        apis.New(apis.Config{Database: resources.Database}),
		KeyService: keys.New(keys.Config{
			Database: resources.Database,
			Events:   events.NewNoop(),
		}),
	})

	res := RemoveApiResponse{}
	testutil.Json(t, srv.app, testutil.JsonRequest{
		Method: "POST",
		Path:   "/v1/apis.removeApi",
		Bearer: resources.UserRootKey,
		Body: fmt.Sprintf(`{
			"apiId":"%s"
			}`, resources.UserApi.Id),
		Response:   &res,
		StatusCode: 200,
	})

	_, found, err := resources.Database.FindApi(ctx, resources.UserApi.Id)
	require.NoError(t, err)
	require.False(t, found)
}
