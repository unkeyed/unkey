package handler

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
)

// TestCanonicalEnvironmentGrantReturnsOnlyItsRows guarantees a narrow gateway
// log grant cannot read sibling environments, apps, or projects.
func TestCanonicalEnvironmentGrantReturnsOnlyItsRows(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	allowed := createGatewayEnvironment(t, h, workspaceID, "allowed")
	siblingEnvironment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   allowed.ProjectID,
		AppID:       allowed.AppID,
		Slug:        "sibling-environment",
	})
	siblingApp := createGatewayEnvironmentInProject(t, h, workspaceID, allowed.ProjectID, "sibling-app")
	otherProject := createGatewayEnvironment(t, h, workspaceID, "other-project")

	for _, row := range []schema.FrontlineRequest{
		{WorkspaceID: workspaceID, ProjectID: allowed.ProjectID, AppID: allowed.AppID, EnvironmentID: allowed.ID, Path: "/allowed"},
		{WorkspaceID: workspaceID, ProjectID: siblingEnvironment.ProjectID, AppID: siblingEnvironment.AppID, EnvironmentID: siblingEnvironment.ID, Path: "/sibling-environment"},
		{WorkspaceID: workspaceID, ProjectID: siblingApp.ProjectID, AppID: siblingApp.AppID, EnvironmentID: siblingApp.ID, Path: "/sibling-app"},
		{WorkspaceID: workspaceID, ProjectID: otherProject.ProjectID, AppID: otherProject.AppID, EnvironmentID: otherProject.ID, Path: "/other-project"},
	} {
		bufferRequest(t, h, row)
	}

	permission := fmt.Sprintf(
		"unkey:v1:%s:projects/%s/apps/%s/environments/%s/gateway/logs#read",
		workspaceID,
		allowed.ProjectID,
		allowed.AppID,
		allowed.ID,
	)
	rootKey := h.CreateRootKey(workspaceID, permission)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
			Query: "SELECT path FROM gateway_requests_v1 ORDER BY path",
		})
		require.Equal(c, 200, res.Status, "response: %s", res.RawBody)
		require.Equal(c, []map[string]any{{"path": "/allowed"}}, res.Body.Data)
	}, 30*time.Second, time.Second)
}

// TestCanonicalGrantUnionPreservesAncestry guarantees project, app, and
// environment grants form an OR union without permitting cross-combinations.
func TestCanonicalGrantUnionPreservesAncestry(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	projectEnvironment := createGatewayEnvironment(t, h, workspaceID, "project-scope")
	projectSibling := createGatewayEnvironmentInProject(t, h, workspaceID, projectEnvironment.ProjectID, "project-sibling")
	appEnvironment := createGatewayEnvironment(t, h, workspaceID, "app-scope")
	appSibling := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   appEnvironment.ProjectID,
		AppID:       appEnvironment.AppID,
		Slug:        "app-sibling",
	})
	forbiddenApp := createGatewayEnvironmentInProject(t, h, workspaceID, appEnvironment.ProjectID, "forbidden-app")
	exactEnvironment := createGatewayEnvironment(t, h, workspaceID, "exact-scope")
	forbiddenExactSibling := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   exactEnvironment.ProjectID,
		AppID:       exactEnvironment.AppID,
		Slug:        "forbidden-exact-sibling",
	})

	for path, environment := range map[string]db.Environment{
		"/allowed-project":       projectEnvironment,
		"/allowed-project-app":   projectSibling,
		"/allowed-app":           appEnvironment,
		"/allowed-app-sibling":   appSibling,
		"/allowed-exact":         exactEnvironment,
		"/forbidden-app":         forbiddenApp,
		"/forbidden-environment": forbiddenExactSibling,
	} {
		bufferRequest(t, h, schema.FrontlineRequest{
			WorkspaceID: workspaceID, ProjectID: environment.ProjectID, AppID: environment.AppID,
			EnvironmentID: environment.ID, Path: path,
		})
	}

	rootKey := h.CreateRootKey(workspaceID,
		fmt.Sprintf("unkey:v1:%s:projects/%s/**#read", workspaceID, projectEnvironment.ProjectID),
		fmt.Sprintf("unkey:v1:%s:projects/%s/apps/%s/**#read", workspaceID, appEnvironment.ProjectID, appEnvironment.AppID),
		fmt.Sprintf("unkey:v1:%s:projects/%s/apps/%s/environments/%s/gateway/logs#read", workspaceID, exactEnvironment.ProjectID, exactEnvironment.AppID, exactEnvironment.ID),
	)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
			Query: "SELECT path FROM gateway_requests_v1 ORDER BY path",
		})
		require.Equal(c, 200, res.Status, "response: %s", res.RawBody)
		require.Equal(c, []map[string]any{
			{"path": "/allowed-app"},
			{"path": "/allowed-app-sibling"},
			{"path": "/allowed-exact"},
			{"path": "/allowed-project"},
			{"path": "/allowed-project-app"},
		}, res.Body.Data)
	}, 30*time.Second, time.Second)
}

// TestCanonicalWorkspaceWideGrantsReturnWorkspaceRows guarantees global and
// gateway-wide grants remain workspace-isolated without extra route filters.
func TestCanonicalWorkspaceWideGrantsReturnWorkspaceRows(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	otherWorkspace := h.CreateWorkspace()
	first := createGatewayEnvironment(t, h, workspaceID, "first")
	second := createGatewayEnvironment(t, h, workspaceID, "second")
	foreign := createGatewayEnvironment(t, h, otherWorkspace.ID, "foreign")
	for path, environment := range map[string]db.Environment{
		"/first":   first,
		"/second":  second,
		"/foreign": foreign,
	} {
		bufferRequest(t, h, schema.FrontlineRequest{
			WorkspaceID: environment.WorkspaceID, ProjectID: environment.ProjectID, AppID: environment.AppID,
			EnvironmentID: environment.ID, Path: path,
		})
	}

	for name, permission := range map[string]string{
		"global administrator": fmt.Sprintf("unkey:v1:%s:**#*", workspaceID),
		"global read":          fmt.Sprintf("unkey:v1:%s:**#read", workspaceID),
		"all projects":         fmt.Sprintf("unkey:v1:%s:projects/*/**#read", workspaceID),
		"all gateway logs":     fmt.Sprintf("unkey:v1:%s:projects/*/apps/*/environments/*/gateway/logs#read", workspaceID),
	} {
		t.Run(name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspaceID, permission)
			require.EventuallyWithT(t, func(c *assert.CollectT) {
				res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
					Query: "SELECT path FROM gateway_requests_v1 ORDER BY path",
				})
				require.Equal(c, 200, res.Status, "response: %s", res.RawBody)
				require.Equal(c, []map[string]any{{"path": "/first"}, {"path": "/second"}}, res.Body.Data)
			}, 30*time.Second, time.Second)
		})
	}
}

// TestCanonicalEmptyResolutionReturnsNoRows guarantees nonexistent and
// contradictory scopes never fall back to unfiltered workspace SQL.
func TestCanonicalEmptyResolutionReturnsNoRows(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	environment := createGatewayEnvironment(t, h, workspaceID, "existing")
	otherWorkspace := h.CreateWorkspace()
	foreignEnvironment := createGatewayEnvironment(t, h, otherWorkspace.ID, "foreign")
	bufferRequest(t, h, schema.FrontlineRequest{
		WorkspaceID: workspaceID, ProjectID: environment.ProjectID, AppID: environment.AppID,
		EnvironmentID: environment.ID, Path: "/must-stay-hidden",
	})

	for name, permission := range map[string]string{
		"nonexistent environment": fmt.Sprintf(
			"unkey:v1:%s:projects/%s/apps/%s/environments/%s/gateway/logs#read",
			workspaceID, environment.ProjectID, environment.AppID, uid.New(uid.EnvironmentPrefix),
		),
		"environment claimed by wrong project": fmt.Sprintf(
			"unkey:v1:%s:projects/%s/apps/%s/environments/%s/gateway/logs#read",
			workspaceID, uid.New(uid.ProjectPrefix), environment.AppID, environment.ID,
		),
		"app claimed by wrong project": fmt.Sprintf(
			"unkey:v1:%s:projects/%s/apps/%s/**#read",
			workspaceID, uid.New(uid.ProjectPrefix), environment.AppID,
		),
		"project belongs to another workspace": fmt.Sprintf(
			"unkey:v1:%s:projects/%s/**#read",
			workspaceID, foreignEnvironment.ProjectID,
		),
	} {
		t.Run(name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspaceID, permission)
			require.EventuallyWithT(t, func(c *assert.CollectT) {
				res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
					Query: "SELECT path FROM gateway_requests_v1",
				})
				require.Equal(c, 200, res.Status, "response: %s", res.RawBody)
				require.Empty(c, res.Body.Data)
			}, 30*time.Second, time.Second)
		})
	}
}

func createGatewayEnvironment(t *testing.T, h *testutil.Harness, workspaceID, name string) db.Environment {
	t.Helper()

	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspaceID,
		Name:        name,
		Slug:        name,
	})
	return createGatewayEnvironmentInProject(t, h, workspaceID, project.ID, name)
}

func createGatewayEnvironmentInProject(t *testing.T, h *testutil.Harness, workspaceID, projectID, name string) db.Environment {
	t.Helper()

	app := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		Name:        name,
		Slug:        name,
	})
	return h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		AppID:       app.ID,
		Slug:        name,
	})
}
