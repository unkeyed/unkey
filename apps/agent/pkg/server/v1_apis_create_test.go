package server_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/server"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
)

func TestV1ApisCreate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := testutil.NewServer(t, resources)

	res := testutil.Json[server.CreateApiResponse](t, srv.App, testutil.JsonRequest{

		Path:       "/v1/apis.createApi",
		Bearer:     resources.UserRootKey,
		Body:       `{ "name":"simple" }`,
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

	srv := testutil.NewServer(t, resources)

	res := testutil.Json[server.ErrorResponse](t, srv.App, testutil.JsonRequest{

		Path:       "/v1/apis.createApi",
		Bearer:     "invalid_key",
		Body:       `{ "name":"simple" }`,
		StatusCode: 401,
	})

	require.Equal(t, "UNAUTHORIZED", res.Error.Code)

}
