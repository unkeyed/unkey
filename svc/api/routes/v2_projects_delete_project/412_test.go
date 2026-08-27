package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_projects_delete_project"
)

func TestDeleteProjectDeleteProtection(t *testing.T) {
	h := testutil.NewHarness(t)
	restate, deletes := newRecordingRestate(t)

	route := &handler.Handler{
		DB:      h.DB,
		Restate: restate,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.delete_project")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	slug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      workspace.ID,
		Name:             "Protected",
		Slug:             slug,
		DeleteProtection: true,
	})

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, headers, handler.Request{
		Project: project.ID,
	})
	require.Equal(t, 412, res.Status, "expected 412, received: %s", res.RawBody)
	require.NotNil(t, res.Body.Error)
	require.Equal(t, "This project has delete protection enabled. Disable it before attempting to delete.", res.Body.Error.Detail)
	testutil.RequireNoReceive(t, deletes, time.Second)
}
