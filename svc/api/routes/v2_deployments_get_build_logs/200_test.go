package handler_test

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_get_build_logs"
)

func TestGetBuildLogsSuccess(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.read_deployment"},
	})
	now := time.Now().UnixMilli()
	firstSeq := uint64(now) * 1000

	call := func(t *testing.T, rootKey string, req handler.Request) *handler.Response {
		t.Helper()
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), req)
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		return res.Body
	}
	messages := func(entries []openapi.BuildLogEntry) []string {
		out := make([]string, 0, len(entries))
		for _, entry := range entries {
			out = append(out, entry.Message)
		}
		return out
	}

	t.Run("first page then the next page by cursor", func(t *testing.T) {
		target := createDeployment(h, setup)
		install := "sha256:" + uid.New("")
		insertStep(t, h, target, install, "[3/4] RUN npm ci", now)
		insertLogs(t, h, target, install, now, firstSeq, 2, false)
		insertLogs(t, h, target, install, now, firstSeq+2, 1, true)

		limit := 2
		first := call(t, setup.RootKey, handler.Request{DeploymentId: target.deploymentID, Limit: &limit})
		require.Equal(t, []openapi.BuildLogEntry{
			{Time: now, StepId: install, Step: "[3/4] RUN npm ci", Output: openapi.BuildLogOutputStdout, Message: "KEBAP 0"},
			{Time: now, StepId: install, Step: "[3/4] RUN npm ci", Output: openapi.BuildLogOutputStdout, Message: "KEBAP 1"},
		}, first.Data)
		require.True(t, first.Pagination.HasMore)
		require.NotNil(t, first.Pagination.Cursor)
		require.Equal(t, strconv.FormatUint(firstSeq+1, 10), *first.Pagination.Cursor)

		next := call(t, setup.RootKey, handler.Request{DeploymentId: target.deploymentID, Cursor: first.Pagination.Cursor, Limit: &limit})
		require.Equal(t, []openapi.BuildLogEntry{
			{Time: now, StepId: install, Step: "[3/4] RUN npm ci", Output: openapi.BuildLogOutputStderr, Message: "KEBAP 0"},
		}, next.Data)
		require.False(t, next.Pagination.HasMore)
		require.NotNil(t, next.Pagination.Cursor)
		require.NotEqual(t, *first.Pagination.Cursor, *next.Pagination.Cursor)
	})

	t.Run("an empty poll echoes the cursor", func(t *testing.T) {
		target := createDeployment(h, setup)
		stepID := "sha256:" + uid.New("")
		insertStep(t, h, target, stepID, "[3/4] RUN npm ci", now)
		insertLogs(t, h, target, stepID, now, firstSeq, 1, false)

		first := call(t, setup.RootKey, handler.Request{DeploymentId: target.deploymentID})
		require.Len(t, first.Data, 1)

		poll := call(t, setup.RootKey, handler.Request{DeploymentId: target.deploymentID, Cursor: first.Pagination.Cursor})
		require.Empty(t, poll.Data)
		require.False(t, poll.Pagination.HasMore)
		require.Equal(t, first.Pagination.Cursor, poll.Pagination.Cursor)
	})

	t.Run("a build without entries has no cursor", func(t *testing.T) {
		target := createDeployment(h, setup)
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{DeploymentId: target.deploymentID})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.Contains(t, res.RawBody, `"data":[]`)
		require.Nil(t, res.Body.Pagination.Cursor)
		require.False(t, res.Body.Pagination.HasMore)
	})

	t.Run("filters by step", func(t *testing.T) {
		target := createDeployment(h, setup)
		install, build := "sha256:"+uid.New(""), "sha256:"+uid.New("")
		insertStep(t, h, target, install, "[3/4] RUN npm ci", now)
		insertStep(t, h, target, build, "[4/4] RUN npm run build", now)
		insertLogs(t, h, target, install, now, firstSeq, 2, false)
		insertLogs(t, h, target, build, now, firstSeq+2, 1, false)

		res := call(t, setup.RootKey, handler.Request{DeploymentId: target.deploymentID, StepId: &build})
		require.Equal(t, []string{"KEBAP 0"}, messages(res.Data))
		require.Equal(t, build, res.Data[0].StepId)
		require.Equal(t, "[4/4] RUN npm run build", res.Data[0].Step)
	})

	t.Run("excludes entries of other deployments", func(t *testing.T) {
		target := createDeployment(h, setup)
		other := createDeployment(h, setup)
		stepID := "sha256:" + uid.New("")
		insertStep(t, h, target, stepID, "[3/4] RUN npm ci", now)
		insertStep(t, h, other, stepID, "[3/4] RUN npm ci", now)
		insertLogs(t, h, target, stepID, now, firstSeq, 1, false)
		insertLogs(t, h, other, stepID, now, firstSeq, 3, false)

		res := call(t, setup.RootKey, handler.Request{DeploymentId: target.deploymentID})
		require.Len(t, res.Data, 1)
	})

	t.Run("permissions", func(t *testing.T) {
		target := createDeployment(h, setup)
		environment := urn.New().Workspace(setup.Workspace.ID).Project(setup.Project.ID).App(setup.App.ID).Environment(setup.Environment.ID)
		for _, tc := range []struct {
			name       string
			permission string
		}{
			{name: "any environment", permission: "environment.*.read_deployment"},
			{name: "this environment", permission: fmt.Sprintf("environment.%s.read_deployment", setup.Environment.ID)},
			{name: "urn for this deployment", permission: rbac.U(environment.Deployment(target.deploymentID), permissions.Read).Value},
			{name: "urn for every deployment", permission: rbac.U(urn.New().Workspace(setup.Workspace.ID).Project("*").App("*").Environment("*").Deployment("*"), permissions.Read).Value},
		} {
			t.Run(tc.name, func(t *testing.T) {
				call(t, h.CreateRootKey(setup.Workspace.ID, tc.permission), handler.Request{DeploymentId: target.deploymentID})
			})
		}
	})
}
