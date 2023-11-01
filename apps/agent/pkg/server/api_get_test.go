package server_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"

	"github.com/unkeyed/unkey/apps/agent/pkg/server"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

func TestGetApi_Exists(t *testing.T) {
	t.Parallel()

	resources := testutil.SetupResources(t)

	srv := testutil.NewServer(t, resources)

	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/apis/%s", resources.UserApi.ApiId), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UserRootKey))

	res, err := srv.App.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)

	successResponse := server.GetApiResponse{}
	err = json.Unmarshal(body, &successResponse)
	require.NoError(t, err)

	require.Equal(t, resources.UserApi.ApiId, successResponse.Id)
	require.Equal(t, resources.UserApi.Name, successResponse.Name)
	require.Equal(t, resources.UserApi.WorkspaceId, successResponse.WorkspaceId)

}

func TestGetApi_NotFound(t *testing.T) {
	t.Parallel()
	resources := testutil.SetupResources(t)

	srv := testutil.NewServer(t, resources)

	fakeApiId := uid.Api()

	res := testutil.Json[server.ErrorResponse](t, srv.App, testutil.JsonRequest{
		Method:     "GET",
		Path:       fmt.Sprintf("/v1/apis/%s", fakeApiId),
		Bearer:     resources.UserRootKey,
		StatusCode: 404,
	})

	require.Equal(t, "NOT_FOUND", res.Error.Code)
	require.Equal(t, fmt.Sprintf("unable to find api: %s", fakeApiId), res.Error.Message)

}

func TestGetApi_WithIpWhitelist(t *testing.T) {
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

	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/apis/%s", api.ApiId), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UserRootKey))

	res, err := srv.App.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)

	successResponse := server.GetApiResponse{}
	err = json.Unmarshal(body, &successResponse)
	require.NoError(t, err)

	require.Equal(t, api.ApiId, successResponse.Id)
	require.Equal(t, api.Name, successResponse.Name)
	require.Equal(t, api.WorkspaceId, successResponse.WorkspaceId)
	require.Equal(t, api.IpWhitelist, successResponse.IpWhitelist)

}
