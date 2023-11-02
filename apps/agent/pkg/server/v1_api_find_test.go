package server_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"

	"github.com/unkeyed/unkey/apps/agent/pkg/server"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

func TestV1FindApi_Exists(t *testing.T) {
	t.Parallel()

	resources := testutil.SetupResources(t)

	srv := testutil.NewServer(t, resources)

	res := testutil.Json[server.GetApiResponseV1](t, srv.App, testutil.JsonRequest{

		Path:       fmt.Sprintf("/v1/apis.getApi?apiId=%s", resources.UserApi.ApiId),
		Bearer:     resources.UserRootKey,
		StatusCode: 200,
	})

	require.Equal(t, resources.UserApi.ApiId, res.Id)
	require.Equal(t, resources.UserApi.Name, res.Name)
	require.Equal(t, resources.UserApi.WorkspaceId, res.WorkspaceId)

}

func TestV1FindApi_NotFound(t *testing.T) {
	t.Parallel()
	resources := testutil.SetupResources(t)

	srv := testutil.NewServer(t, resources)

	fakeApiId := uid.Api()

	res := testutil.Json[server.ErrorResponse](t, srv.App, testutil.JsonRequest{

		Path:       fmt.Sprintf("/v1/apis?apiId=%s", fakeApiId),
		Bearer:     resources.UserRootKey,
		StatusCode: 404,
	})

	require.Equal(t, "NOT_FOUND", res.Error.Code)
	require.Equal(t, fmt.Sprintf("api %s does not exist", fakeApiId), res.Error.Message)

}

func TestV1FindApi_WithIpWhitelist(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	resources := testutil.SetupResources(t)

	srv := testutil.NewServer(t, resources)

	keyAuth := &authenticationv1.KeyAuth{
		KeyAuthId:   uid.KeyAuth(),
		WorkspaceId: resources.UserWorkspace.WorkspaceId,
	}
	err := resources.Database.InsertKeyAuth(ctx, keyAuth)
	require.NoError(t, err)

	api := &apisv1.Api{
		ApiId:       uid.Api(),
		Name:        "test",
		WorkspaceId: resources.UserWorkspace.WorkspaceId,
		IpWhitelist: []string{"127.0.0.1", "1.1.1.1"},
		AuthType:    apisv1.AuthType_AUTH_TYPE_KEY,
		KeyAuthId:   &keyAuth.KeyAuthId,
	}

	err = resources.Database.InsertApi(ctx, api)
	require.NoError(t, err)

	res := testutil.Json[server.GetApiResponseV1](t, srv.App, testutil.JsonRequest{

		Path:       fmt.Sprintf("/v1/apis.getApi?apiId=%s", api.ApiId),
		Bearer:     resources.UserRootKey,
		StatusCode: 200,
	})

	require.Equal(t, api.ApiId, res.Id)
	require.Equal(t, api.Name, res.Name)
	require.Equal(t, api.WorkspaceId, res.WorkspaceId)
	require.Equal(t, api.IpWhitelist, res.IpWhitelist)

}
