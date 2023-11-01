package server_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/server"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

func TestCreateWorkspace_Simple(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resources := testutil.SetupResources(t)

	srv := testutil.NewServer(t, resources)

	tenantId := uid.New(16, "")
	res := testutil.Json[server.CreateWorkspaceResponseV1](t, srv.App, testutil.JsonRequest{
		Method: "POST",
		Path:   "/v1/workspaces.createWorkspace",
		Body: fmt.Sprintf(`{
			"name":"simple",
			"tenantId": "%s"
			}`, tenantId),
		Bearer:     resources.UnkeyAppAuthToken,
		StatusCode: 200,
	})
	t.Logf("res: %+v", res)

	require.NotEmpty(t, res.Id)

	foundWorkspace, found, err := resources.Database.FindWorkspace(ctx, res.Id)
	require.NoError(t, err)
	require.True(t, found)

	require.Equal(t, res.Id, foundWorkspace.WorkspaceId)
	require.Equal(t, "simple", foundWorkspace.Name)
	require.Equal(t, tenantId, foundWorkspace.TenantId)
}
