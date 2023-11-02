package server_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
)

func TestV1ApisRemove(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := testutil.NewServer(t, resources)

	testutil.Json[any](t, srv.App, testutil.JsonRequest{
		Path:   "/v1/apis.deleteApi",
		Bearer: resources.UserRootKey,
		Body: fmt.Sprintf(`{
			"apiId":"%s"
			}`, resources.UserApi.ApiId),
		StatusCode: 200,
	})

	_, found, err := resources.Database.FindApi(ctx, resources.UserApi.ApiId)
	require.NoError(t, err)
	require.False(t, found)
}
