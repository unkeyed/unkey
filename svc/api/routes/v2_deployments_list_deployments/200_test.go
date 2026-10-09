package handler_test

import (
	"cmp"
	"context"
	"database/sql"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_list_deployments"
)

func TestListWorkspaceWide(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	setup.RootKey = h.CreateRootKey(setup.Workspace.ID, readDeployments(setup.Workspace.ID))

	want := map[string]bool{}
	for range 3 {
		dep := h.CreateDeployment(seed.CreateDeploymentRequest{
			ID:            uid.New(uid.DeploymentPrefix),
			WorkspaceID:   setup.Workspace.ID,
			ProjectID:     setup.Project.ID,
			AppID:         setup.App.ID,
			EnvironmentID: setup.Environment.ID,
			Source:        db.DeploymentsSourceGit,
			GitBranch:     "main",
			GitCommitSha:  "abc123",
		})
		want[dep.ID] = true
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.NotNil(t, res.Body)
	require.Len(t, res.Body.Data, 3)
	require.False(t, res.Body.Pagination.HasMore)

	for _, d := range res.Body.Data {
		require.True(t, want[d.Id], "unexpected deployment %s", d.Id)
		require.Equal(t, openapi.DeploymentStatusPending, d.Status)

		// Enriched fields present on list items (cheap, batched).
		require.Equal(t, setup.Environment.Slug, d.Environment)
		require.Equal(t, setup.App.Slug, d.App)
		require.Equal(t, setup.Project.Slug, d.Project)
		require.NotNil(t, d.AvailableActions)
		require.Empty(t, d.Regions, "no topology seeded")
		require.NotNil(t, d.Git)
		require.Equal(t, "abc123", d.Git.CommitSha)

		// error is only set for failed deployments; these are pending.
		require.Nil(t, d.Error, "pending deployments have no error")
		// domains are always present as a slice, empty when none are configured.
		require.NotNil(t, d.Domains)
		require.Empty(t, *d.Domains)
	}

	// Internal fields must never appear in the response body.
	for _, leaked := range []string{"k8s_name", "k8sName", "workspace_id", "workspaceId", "sentinel", "encrypted", "build_id", "buildId", "invocation", "github_deployment", "githubDeployment", "\"pk\""} {
		require.False(t, strings.Contains(res.RawBody, leaked), "response leaked internal field %q: %s", leaked, res.RawBody)
	}
}

func TestListFilterByEnvironment(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	setup.RootKey = h.CreateRootKey(setup.Workspace.ID, readDeployments(setup.Workspace.ID))

	// A second environment in the same app whose deployments must be excluded.
	otherEnv := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   setup.Project.ID,
		AppID:       setup.App.ID,
		Slug:        "preview",
		Description: "preview environment",
	})

	target := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})
	h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: otherEnv.ID,
	})

	req := handler.Request{
		Project:     rid(setup.Project.Slug),
		App:         rid(setup.App.Slug),
		Environment: rid(setup.Environment.Slug),
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, res.Body.Data, 1)
	require.Equal(t, target.ID, res.Body.Data[0].Id)
}

func TestListFilterByProject(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	setup.RootKey = h.CreateRootKey(setup.Workspace.ID, readDeployments(setup.Workspace.ID))

	target := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})

	// A second project in the same workspace whose deployments must be excluded.
	otherProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: setup.Workspace.ID,
		Name:        "other",
		Slug:        "other-project",
	})
	otherApp := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   otherProject.ID,
		Name:        "other",
		Slug:        "other-app",
	})
	otherEnv := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   otherProject.ID,
		AppID:       otherApp.ID,
		Slug:        "production",
		Kind:        mysqltype.EnvironmentKindProduction,
		Description: "other project production",
	})
	h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     otherProject.ID,
		AppID:         otherApp.ID,
		EnvironmentID: otherEnv.ID,
	})

	// Resolve the project by id to exercise the id-seek branch of the resolver.
	req := handler.Request{Project: rid(setup.Project.ID)}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, res.Body.Data, 1)
	require.Equal(t, target.ID, res.Body.Data[0].Id)
}

func TestListFilterByApp(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	setup.RootKey = h.CreateRootKey(setup.Workspace.ID, readDeployments(setup.Workspace.ID))

	// A second app in the same project whose deployments must be excluded.
	otherApp := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   setup.Project.ID,
		Name:        "other",
		Slug:        "other-app",
	})
	otherEnv := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   setup.Project.ID,
		AppID:       otherApp.ID,
		Slug:        "production",
		Kind:        mysqltype.EnvironmentKindProduction,
		Description: "other app production",
	})

	target := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})
	h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         otherApp.ID,
		EnvironmentID: otherEnv.ID,
	})

	// project + app filter, no environment: app_id resolves, environment_id is NULL.
	req := handler.Request{
		Project: rid(setup.Project.Slug),
		App:     rid(setup.App.Slug),
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, res.Body.Data, 1)
	require.Equal(t, target.ID, res.Body.Data[0].Id)
}

func TestListFilterByStatus(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	setup.RootKey = h.CreateRootKey(setup.Workspace.ID, readDeployments(setup.Workspace.ID))

	// Seeded deployments are all in status "pending".
	h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})

	pending := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		Status: new([]openapi.DeploymentStatus{openapi.DeploymentStatusPending}),
	})
	require.Equal(t, http.StatusOK, pending.Status, "expected 200, received: %s", pending.RawBody)
	require.Len(t, pending.Body.Data, 1)

	failed := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		Status: new([]openapi.DeploymentStatus{openapi.DeploymentStatusFailed}),
	})
	require.Equal(t, http.StatusOK, failed.Status, "expected 200, received: %s", failed.RawBody)
	require.Empty(t, failed.Body.Data)
}

// An explicit empty status array disables the filter (same as omitting it): the
// has_status_filter gate keeps the empty IN (NULL) from matching nothing.
func TestListEmptyStatusFilter(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	setup.RootKey = h.CreateRootKey(setup.Workspace.ID, readDeployments(setup.Workspace.ID))

	const total = 3
	for range total {
		h.CreateDeployment(seed.CreateDeploymentRequest{
			ID:            uid.New(uid.DeploymentPrefix),
			WorkspaceID:   setup.Workspace.ID,
			ProjectID:     setup.Project.ID,
			AppID:         setup.App.ID,
			EnvironmentID: setup.Environment.ID,
		})
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		Status: new([]openapi.DeploymentStatus{}),
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, res.Body.Data, total, "empty status array must not filter anything out")
}

// A multi-value status filter returns deployments in any of the listed statuses
// and excludes the rest.
func TestListFilterByMultipleStatuses(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	setup.RootKey = h.CreateRootKey(setup.Workspace.ID, readDeployments(setup.Workspace.ID))

	pending := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
		Status:        mysqltype.DeploymentsStatusPending,
	})
	failed := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
		Status:        mysqltype.DeploymentsStatusFailed,
	})
	// A ready deployment that must be excluded by the filter below.
	h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
		Status:        mysqltype.DeploymentsStatusReady,
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		Status: new([]openapi.DeploymentStatus{openapi.DeploymentStatusPending, openapi.DeploymentStatusFailed}),
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, res.Body.Data, 2)

	got := map[string]bool{}
	for _, d := range res.Body.Data {
		got[d.Id] = true
	}
	require.True(t, got[pending.ID], "pending deployment missing from multi-status result")
	require.True(t, got[failed.ID], "failed deployment missing from multi-status result")
}

func TestListEmptyWorkspace(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	setup.RootKey = h.CreateRootKey(setup.Workspace.ID, readDeployments(setup.Workspace.ID))

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Empty(t, res.Body.Data)
	require.False(t, res.Body.Pagination.HasMore)
}

// Deployments are returned newest-first. pk is assigned in insertion order, so
// the response must be the reverse of the order they were created in.
func TestListNewestFirst(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	setup.RootKey = h.CreateRootKey(setup.Workspace.ID, readDeployments(setup.Workspace.ID))

	const total = 5
	insertionOrder := make([]string, 0, total)
	for range total {
		dep := h.CreateDeployment(seed.CreateDeploymentRequest{
			ID:            uid.New(uid.DeploymentPrefix),
			WorkspaceID:   setup.Workspace.ID,
			ProjectID:     setup.Project.ID,
			AppID:         setup.App.ID,
			EnvironmentID: setup.Environment.ID,
		})
		insertionOrder = append(insertionOrder, dep.ID)
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, res.Body.Data, total)

	want := make([]string, total)
	for i, id := range insertionOrder {
		want[total-1-i] = id
	}
	got := make([]string, total)
	for i, d := range res.Body.Data {
		got[i] = d.Id
	}
	require.Equal(t, want, got, "deployments must be returned newest-first")
}

// A workspace-wide list must only return the caller's deployments; another
// workspace's deployments must never leak in.
func TestListWorkspaceIsolation(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	caller := h.CreateTestDeploymentSetup()
	caller.RootKey = h.CreateRootKey(caller.Workspace.ID, readDeployments(caller.Workspace.ID))
	other := h.CreateTestDeploymentSetup()

	mine := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   caller.Workspace.ID,
		ProjectID:     caller.Project.ID,
		AppID:         caller.App.ID,
		EnvironmentID: caller.Environment.ID,
	})
	h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   other.Workspace.ID,
		ProjectID:     other.Project.ID,
		AppID:         other.App.ID,
		EnvironmentID: other.Environment.ID,
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(caller.RootKey), handler.Request{})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, res.Body.Data, 1)
	require.Equal(t, mine.ID, res.Body.Data[0].Id)
}

// Paging with a small limit must walk the cursor across every page, returning
// each deployment exactly once with no gaps or duplicates, and stop with a nil
// cursor when hasMore is false.
func TestListPagination(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	setup.RootKey = h.CreateRootKey(setup.Workspace.ID, readDeployments(setup.Workspace.ID))

	const total = 5
	created := map[string]bool{}
	for range total {
		dep := h.CreateDeployment(seed.CreateDeploymentRequest{
			ID:            uid.New(uid.DeploymentPrefix),
			WorkspaceID:   setup.Workspace.ID,
			ProjectID:     setup.Project.ID,
			AppID:         setup.App.ID,
			EnvironmentID: setup.Environment.ID,
		})
		created[dep.ID] = true
	}

	seen := map[string]bool{}
	var cursor *string
	pages := 0
	for {
		req := handler.Request{Limit: new(2), Cursor: cursor}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.LessOrEqual(t, len(res.Body.Data), 2)

		for _, d := range res.Body.Data {
			require.False(t, seen[d.Id], "deployment %s returned on more than one page", d.Id)
			seen[d.Id] = true
		}

		pages++
		require.Less(t, pages, 10, "pagination did not terminate")

		if !res.Body.Pagination.HasMore {
			require.Nil(t, res.Body.Pagination.Cursor)
			break
		}
		require.NotNil(t, res.Body.Pagination.Cursor)
		cursor = res.Body.Pagination.Cursor
	}

	require.Len(t, seen, total)
	for id := range created {
		require.True(t, seen[id], "deployment %s missing from paginated results", id)
	}
}

// Pages follow creation time, not insertion order, so a deployment inserted
// late with an older createdAt still lands where the dashboard sorts it. Ties
// on createdAt fall back to insertion order, newest first.
func TestListPaginationFollowsCreatedAt(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	setup.RootKey = h.CreateRootKey(setup.Workspace.ID, readDeployments(setup.Workspace.ID))

	base := time.Now().UnixMilli()
	type inserted struct {
		id        string
		createdAt int64
		order     int
	}
	var rows []inserted
	for i, offset := range []int64{300, 100, 500, 200, 200, 400} {
		dep := h.CreateDeployment(seed.CreateDeploymentRequest{
			ID:            uid.New(uid.DeploymentPrefix),
			WorkspaceID:   setup.Workspace.ID,
			ProjectID:     setup.Project.ID,
			AppID:         setup.App.ID,
			EnvironmentID: setup.Environment.ID,
			CreatedAt:     base + offset,
		})
		rows = append(rows, inserted{id: dep.ID, createdAt: base + offset, order: i})
	}
	slices.SortFunc(rows, func(a, b inserted) int {
		return cmp.Or(cmp.Compare(b.createdAt, a.createdAt), cmp.Compare(b.order, a.order))
	})
	want := make([]string, len(rows))
	for i, r := range rows {
		want[i] = r.id
	}

	var got []string
	var cursor *string
	for range 10 {
		req := handler.Request{Limit: new(2), Cursor: cursor}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		for _, d := range res.Body.Data {
			got = append(got, d.Id)
		}
		if !res.Body.Pagination.HasMore {
			break
		}
		cursor = res.Body.Pagination.Cursor
	}

	require.Equal(t, want, got)
}

func TestListFilterByBranch(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	setup.RootKey = h.CreateRootKey(setup.Workspace.ID, readDeployments(setup.Workspace.ID))

	want := map[string]bool{}
	for _, branch := range []string{"main", "feature/kebap", "release", ""} {
		dep := h.CreateDeployment(seed.CreateDeploymentRequest{
			ID:            uid.New(uid.DeploymentPrefix),
			WorkspaceID:   setup.Workspace.ID,
			ProjectID:     setup.Project.ID,
			AppID:         setup.App.ID,
			EnvironmentID: setup.Environment.ID,
			Source:        db.DeploymentsSourceGit,
			GitBranch:     branch,
			GitCommitSha:  "abc123",
		})
		if branch == "main" || branch == "feature/kebap" {
			want[dep.ID] = true
		}
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		Project: rid(setup.Project.Slug),
		App:     rid(setup.App.Slug),
		Branch:  &[]string{"main", "feature/kebap"},
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, res.Body.Data, len(want))
	for _, d := range res.Body.Data {
		require.True(t, want[d.Id], "deployment %s is not on a requested branch", d.Id)
	}
	empty := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{Branch: new([]string{})})
	require.Equal(t, http.StatusOK, empty.Status, "an empty branch list is no filter, received: %s", empty.RawBody)
	require.Len(t, empty.Body.Data, 4)
}

func TestListFilterByTime(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	setup.RootKey = h.CreateRootKey(setup.Workspace.ID, readDeployments(setup.Workspace.ID))

	const start, end = int64(1_704_067_200_000), int64(1_704_672_000_000)
	byCreatedAt := map[int64]string{}
	for _, createdAt := range []int64{start - 1, start, end - 1, end} {
		dep := h.CreateDeployment(seed.CreateDeploymentRequest{
			ID:            uid.New(uid.DeploymentPrefix),
			WorkspaceID:   setup.Workspace.ID,
			ProjectID:     setup.Project.ID,
			AppID:         setup.App.ID,
			EnvironmentID: setup.Environment.ID,
			CreatedAt:     createdAt,
		})
		byCreatedAt[createdAt] = dep.ID
	}

	list := func(req handler.Request) []string {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		ids := make([]string, len(res.Body.Data))
		for i, d := range res.Body.Data {
			ids[i] = d.Id
		}
		return ids
	}

	t.Run("startTime is inclusive and endTime is exclusive", func(t *testing.T) {
		got := list(handler.Request{StartTime: new(start), EndTime: new(end)})
		require.ElementsMatch(t, []string{byCreatedAt[start], byCreatedAt[end-1]}, got)
	})
	t.Run("startTime alone", func(t *testing.T) {
		got := list(handler.Request{StartTime: new(end)})
		require.ElementsMatch(t, []string{byCreatedAt[end]}, got)
	})
	t.Run("endTime alone", func(t *testing.T) {
		got := list(handler.Request{EndTime: new(start)})
		require.ElementsMatch(t, []string{byCreatedAt[start-1]}, got)
	})
}

func TestListDeploymentFields(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	setup.RootKey = h.CreateRootKey(setup.Workspace.ID, readDeployments(setup.Workspace.ID))

	recordStep := func(deploymentID string, step db.DeploymentStepsStep, endedAt int64) {
		ctx := context.Background()
		require.NoError(t, db.Query.InsertDeploymentStep(ctx, h.DB.RW(), db.InsertDeploymentStepParams{
			WorkspaceID:   setup.Workspace.ID,
			ProjectID:     setup.Project.ID,
			AppID:         setup.App.ID,
			EnvironmentID: setup.Environment.ID,
			DeploymentID:  deploymentID,
			Step:          step,
			StartedAt:     1,
		}))
		if endedAt == 0 {
			return
		}
		require.NoError(t, db.Query.EndDeploymentStep(ctx, h.DB.RW(), db.EndDeploymentStepParams{
			DeploymentID: deploymentID,
			Step:         step,
			EndedAt:      sql.NullInt64{Valid: true, Int64: endedAt},
			Error:        sql.NullString{Valid: false},
		}))
	}

	triggeredBy := uid.New(uid.KeyPrefix)
	gitDep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:                     uid.New(uid.DeploymentPrefix),
		WorkspaceID:            setup.Workspace.ID,
		ProjectID:              setup.Project.ID,
		AppID:                  setup.App.ID,
		EnvironmentID:          setup.Environment.ID,
		Status:                 mysqltype.DeploymentsStatusReady,
		Source:                 db.DeploymentsSourceGit,
		GitBranch:              "main",
		GitCommitSha:           "9f2c1a7",
		GitCommitMessage:       "add KEBAP endpoint",
		GitCommitAuthorHandle:  "octocat",
		GitCommitAuthorAvatar:  "https://avatars.githubusercontent.com/u/1",
		GitCommitTimestamp:     1_704_067_100_000,
		PrNumber:               412,
		ForkRepositoryFullName: "octocat/kebap",
		Trigger:                db.DeploymentsTriggerCli,
		TriggeredBy:            triggeredBy,
	})
	recordStep(gitDep.ID, db.DeploymentStepsStepBuilding, 1_704_067_230_000)
	recordStep(gitDep.ID, db.DeploymentStepsStepDeploying, 1_704_067_260_000)

	ociDep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:             uid.New(uid.DeploymentPrefix),
		WorkspaceID:    setup.Workspace.ID,
		ProjectID:      setup.Project.ID,
		AppID:          setup.App.ID,
		EnvironmentID:  setup.Environment.ID,
		Status:         mysqltype.DeploymentsStatusReady,
		Source:         db.DeploymentsSourceOci,
		ImageRequested: "ghcr.io/acme/kebap:v1",
		ImageResolved:  "ghcr.io/acme/kebap@sha256:abc",
	})
	recordStep(ociDep.ID, db.DeploymentStepsStepQueued, 1_704_067_230_000)
	recordStep(ociDep.ID, db.DeploymentStepsStepDeploying, 0)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, res.Body.Data, 2)
	byID := map[string]openapi.Deployment{}
	for _, d := range res.Body.Data {
		byID[d.Id] = d
	}

	t.Run("git deployment", func(t *testing.T) {
		d := byID[gitDep.ID]
		require.Equal(t, &openapi.DeploymentGit{
			CommitSha:       "9f2c1a7",
			Branch:          new("main"),
			CommitMessage:   new("add KEBAP endpoint"),
			CommitTimestamp: new(int64(1_704_067_100_000)),
			Author: &openapi.DeploymentGitAuthor{
				Handle:    "octocat",
				AvatarUrl: new("https://avatars.githubusercontent.com/u/1"),
			},
			PrNumber:       new(412),
			ForkRepository: new("octocat/kebap"),
		}, d.Git)
		require.Equal(t, openapi.DeploymentTrigger{
			Via:   openapi.DeploymentTriggerViaCli,
			Actor: &openapi.DeploymentTriggerActor{Type: openapi.DeploymentTriggerActorTypeRootKey, Id: triggeredBy},
		}, d.Trigger)
		require.Equal(t, new(int64(1_704_067_260_000)), d.FinishedAt)
	})

	t.Run("oci deployment with an open step", func(t *testing.T) {
		d := byID[ociDep.ID]
		require.Equal(t, &openapi.DeploymentDocker{Image: "ghcr.io/acme/kebap:v1", ResolvedImage: new("ghcr.io/acme/kebap@sha256:abc")}, d.Docker)
		require.Equal(t, openapi.DeploymentTrigger{Via: openapi.DeploymentTriggerViaUnknown, Actor: nil}, d.Trigger)
		require.Nil(t, d.FinishedAt, "a step is still open")
	})
}

// A cursor names a deployment by id, so it must resolve only inside the
// caller's workspace: another workspace's id behaves like an unknown one
func TestListForeignCursor(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	caller := h.CreateTestDeploymentSetup()
	caller.RootKey = h.CreateRootKey(caller.Workspace.ID, readDeployments(caller.Workspace.ID))
	other := h.CreateTestDeploymentSetup()
	h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   caller.Workspace.ID,
		ProjectID:     caller.Project.ID,
		AppID:         caller.App.ID,
		EnvironmentID: caller.Environment.ID,
	})
	foreign := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   other.Workspace.ID,
		ProjectID:     other.Project.ID,
		AppID:         other.App.ID,
		EnvironmentID: other.Environment.ID,
	})

	for _, cursor := range []string{foreign.ID, uid.New(uid.DeploymentPrefix)} {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(caller.RootKey), handler.Request{Cursor: new(cursor)})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.Empty(t, res.Body.Data)
	}
}
