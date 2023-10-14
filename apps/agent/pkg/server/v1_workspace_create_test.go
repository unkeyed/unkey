package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/workspaces"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

func TestCreateWorkspace_Simple(t *testing.T) {
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
	})

	tenantId := uid.New(16, "")
	res := CreateWorkspaceResponseV1{}
	testutil.Json(t, srv.app, testutil.JsonRequest{
		Debug:  true,
		Method: "POST",
		Path:   "/v1/workspaces.createWorkspace",
		Body: fmt.Sprintf(`{
			"name":"simple",
			"tenantId": "%s"
			}`, tenantId),
		Bearer:     srv.unkeyAppAuthToken,
		Response:   &res,
		StatusCode: 200,
	})

	require.NotEmpty(t, res.Id)

	foundWorkspace, found, err := resources.Database.FindWorkspace(ctx, res.Id)
	require.NoError(t, err)
	require.True(t, found)

	require.Equal(t, res.Id, foundWorkspace.Id)
	require.Equal(t, "simple", foundWorkspace.Name)
	require.Equal(t, tenantId, foundWorkspace.TenantId)
}
