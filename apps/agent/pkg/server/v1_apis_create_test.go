package server

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/apis"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/workspaces"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

func TestV1ApisCreate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := New(Config{
		Logger:            logging.NewNoopLogger(),
		KeyCache:          cache.NewNoopCache[entities.Key](),
		ApiCache:          cache.NewNoopCache[entities.Api](),
		Database:          resources.Database,
		Tracer:            tracing.NewNoop(),
		UnkeyWorkspaceId:  resources.UnkeyWorkspace.Id,
		UnkeyApiId:        resources.UnkeyApi.Id,
		UnkeyAppAuthToken: "supersecret",
		WorkspaceService:  workspaces.New(workspaces.Config{Database: resources.Database}),
		ApiService:        apis.New(apis.Config{Database: resources.Database}),
	})

	res := CreateApiResponse{}
	testutil.Json(t, srv.app, testutil.JsonRequest{
		Debug:      true,
		Method:     "POST",
		Path:       "/v1/api.createApi",
		Bearer:     resources.UserRootKey,
		Body:       `{ "name":"simple" }`,
		Response:   &res,
		StatusCode: 200,
	})

	require.NotEmpty(t, res.ApiId)

	foundApi, found, err := resources.Database.FindApi(ctx, res.ApiId)
	require.NoError(t, err)
	require.True(t, found)

	require.Equal(t, res.ApiId, foundApi.Id)
	require.Equal(t, "simple", foundApi.Name)
	require.Equal(t, resources.UserWorkspace.Id, foundApi.WorkspaceId)
}

func TestV1ApisCreate_RejectsUnauthorized(t *testing.T) {
	t.Skip("TODO: implement")
	t.Parallel()

	resources := testutil.SetupResources(t)

	srv := New(Config{
		Logger:            logging.NewNoopLogger(),
		KeyCache:          cache.NewNoopCache[entities.Key](),
		ApiCache:          cache.NewNoopCache[entities.Api](),
		Database:          resources.Database,
		Tracer:            tracing.NewNoop(),
		UnkeyWorkspaceId:  resources.UnkeyWorkspace.Id,
		UnkeyApiId:        resources.UnkeyApi.Id,
		UnkeyAppAuthToken: "supersecret",
		WorkspaceService:  workspaces.New(workspaces.Config{Database: resources.Database}),
		ApiService:        apis.New(apis.Config{Database: resources.Database}),
	})

	t.Logf("%+v", resources)
	res := errors.ErrorResponse{}
	testutil.Json(t, srv.app, testutil.JsonRequest{
		Debug:      true,
		Method:     "POST",
		Path:       "/v1/api.createApi",
		Bearer:     resources.UserRootKey,
		Body:       `{ "name":"simple" }`,
		Response:   &res,
		StatusCode: 400,
	})

	require.Equal(t, "", res.Error)

}
