package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/workspaces"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
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

	buf := bytes.NewBufferString(`{
		"name":"simple",
		"tenantId": "user_123"
		}`)

	req := httptest.NewRequest("POST", "/v1/workspaces.create", buf)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", srv.unkeyAppAuthToken))
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	t.Logf("res: %s", string(body))
	require.Equal(t, res.StatusCode, 200)

	createWorkspaceResponse := CreateWorkspaceResponse{}
	err = json.Unmarshal(body, &createWorkspaceResponse)
	require.NoError(t, err)

	require.NotEmpty(t, createWorkspaceResponse.Id)

	foundWorkspace, found, err := resources.Database.FindWorkspace(ctx, createWorkspaceResponse.Id)
	require.NoError(t, err)
	require.True(t, found)

	require.Equal(t, createWorkspaceResponse.Id, foundWorkspace.Id)
	require.Equal(t, "simple", foundWorkspace.Name)
	require.Equal(t, "user_123", foundWorkspace.TenantId)
}
