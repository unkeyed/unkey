package server

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/apis"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/keys"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/workspaces"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

func TestV1ApisCreate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := New(Config{
		Logger:            logging.NewNoop(),
		KeyCache:          cache.NewNoopCache[*authenticationv1.Key](),
		ApiCache:          cache.NewNoopCache[*apisv1.Api](),
		Database:          resources.Database,
		Tracer:            tracing.NewNoop(),
		UnkeyWorkspaceId:  resources.UnkeyWorkspace.WorkspaceId,
		UnkeyApiId:        resources.UnkeyApi.ApiId,
		UnkeyAppAuthToken: "supersecret",
		WorkspaceService:  workspaces.New(workspaces.Config{Database: resources.Database}),
		ApiService:        apis.New(apis.Config{Database: resources.Database}),
		KeyService: keys.New(keys.Config{
			Database: resources.Database,
			Events:   events.NewNoop(),
		}),
	})

	res := CreateApiResponse{}
	testutil.Json(t, srv.app, testutil.JsonRequest{
		Method:     "POST",
		Path:       "/v1/apis.createApi",
		Bearer:     resources.UserRootKey,
		Body:       `{ "name":"simple" }`,
		Response:   &res,
		StatusCode: 200,
	})

	require.NotEmpty(t, res.ApiId)

	foundApi, found, err := resources.Database.FindApi(ctx, res.ApiId)
	require.NoError(t, err)
	require.True(t, found)

	require.Equal(t, res.ApiId, foundApi.ApiId)
	require.Equal(t, "simple", foundApi.Name)
	require.Equal(t, resources.UserWorkspace.WorkspaceId, foundApi.WorkspaceId)
}

func TestV1ApisCreate_RejectsUnauthorized(t *testing.T) {
	t.Parallel()

	resources := testutil.SetupResources(t)

	srv := New(Config{
		Logger:            logging.NewNoop(),
		KeyCache:          cache.NewNoopCache[*authenticationv1.Key](),
		ApiCache:          cache.NewNoopCache[*apisv1.Api](),
		Database:          resources.Database,
		Tracer:            tracing.NewNoop(),
		UnkeyWorkspaceId:  resources.UnkeyWorkspace.WorkspaceId,
		UnkeyApiId:        resources.UnkeyApi.ApiId,
		UnkeyAppAuthToken: "supersecret",
		WorkspaceService:  workspaces.New(workspaces.Config{Database: resources.Database}),
		ApiService:        apis.New(apis.Config{Database: resources.Database}),
	})

	res := errors.ErrorResponse{}
	testutil.Json(t, srv.app, testutil.JsonRequest{
		Method:     "POST",
		Path:       "/v1/apis.createApi",
		Bearer:     "invalid_key",
		Body:       `{ "name":"simple" }`,
		Response:   &res,
		StatusCode: 401,
	})

	require.Equal(t, "UNAUTHORIZED", res.Error.Code)

}
