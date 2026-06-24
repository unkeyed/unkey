package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_projects_create_project"
)

func TestCreateProjectDuplicate(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		CtrlClient: &testutil.MockProjectClient{
			CreateProjectFunc: func(ctx context.Context, req *ctrlv1.CreateProjectRequest) (*ctrlv1.CreateProjectResponse, error) {
				return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("project with slug %q already exists", req.GetSlug()))
			},
		},
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "project.*.create_project")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("duplicate slug returns 409", func(t *testing.T) {
		req := handler.Request{Name: "Payments Service", Slug: "duplicate-slug"}

		errorRes := testutil.CallRoute[handler.Request, openapi.ConflictErrorResponse](h, route, headers, req)
		require.Equal(t, 409, errorRes.Status, "expected 409, received: %s", errorRes.RawBody)
		require.NotNil(t, errorRes.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/project_already_exists", errorRes.Body.Error.Type)
	})
}
