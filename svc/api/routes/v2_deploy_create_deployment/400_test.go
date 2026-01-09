package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_create_deployment"
)

func TestBadRequests(t *testing.T) {
	h := testutil.NewHarness(t)

	// Get CTRL service URL and token
	ctrlURL, ctrlToken := containers.ControlPlane(t)

	ctrlClient := ctrlv1connect.NewDeploymentServiceClient(
		http.DefaultClient,
		ctrlURL,
	)

	route := &handler.Handler{
		Logger:     h.Logger,
		DB:         h.DB,
		Keys:       h.Keys,
		CtrlClient: ctrlClient,
		CtrlToken:  ctrlToken,
	}
	h.Register(route)

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID)

	projectID := uid.New(uid.ProjectPrefix)
	projectName := "test-project"
	projectSlug := "production"

	project := h.CreateProject(seed.CreateProjectRequest{
		WorkspaceID: workspace.ID,
		Name:        projectName,
		ID:          projectID,
		Slug:        projectSlug,
	})

	h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:               uid.New(uid.EnvironmentPrefix),
		WorkspaceID:      workspace.ID,
		ProjectID:        project.ID,
		Slug:             projectSlug,
		Description:      "Production environment",
		DeleteProtection: false,
	})

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("missing both buildContext and dockerImage", func(t *testing.T) {
		req := handler.Request{
			ProjectId:       project.ID,
			Branch:          "main",
			EnvironmentSlug: "production",
			// Neither buildContext nor dockerImage provided
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "Either buildContext or dockerImage must be provided.", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("both buildContext and dockerImage provided", func(t *testing.T) {
		req := handler.Request{
			ProjectId:       project.ID,
			Branch:          "main",
			EnvironmentSlug: "production",
			BuildContext: &struct {
				BuildContextPath string  `json:"buildContextPath"`
				DockerfilePath   *string `json:"dockerfilePath,omitempty"`
			}{
				BuildContextPath: "/app",
			},
			DockerImage: ptr.P("nginx:latest"),
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "Only one of buildContext or dockerImage can be provided.", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("missing projectId", func(t *testing.T) {
		req := handler.Request{
			Branch:          "main",
			EnvironmentSlug: "production",
			DockerImage:     ptr.P("nginx:latest"),
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
	})

	t.Run("missing branch", func(t *testing.T) {
		req := handler.Request{
			ProjectId:       project.ID,
			EnvironmentSlug: "production",
			DockerImage:     ptr.P("nginx:latest"),
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Greater(t, len(res.Body.Error.Errors), 0)
	})

	t.Run("missing environmentSlug", func(t *testing.T) {
		req := handler.Request{
			ProjectId:   project.ID,
			Branch:      "main",
			DockerImage: ptr.P("nginx:latest"),
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Greater(t, len(res.Body.Error.Errors), 0)
	})

	t.Run("missing authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type": {"application/json"},
			// No Authorization header
		}

		req := handler.Request{
			ProjectId:       project.ID,
			Branch:          "main",
			EnvironmentSlug: "production",
			DockerImage:     ptr.P("nginx:latest"),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("malformed authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"malformed_header"},
		}

		req := handler.Request{
			ProjectId:       project.ID,
			Branch:          "main",
			EnvironmentSlug: "production",
			DockerImage:     ptr.P("nginx:latest"),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("empty projectId", func(t *testing.T) {
		req := handler.Request{
			ProjectId:       "",
			Branch:          "main",
			EnvironmentSlug: "production",
			DockerImage:     ptr.P("nginx:latest"),
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("empty branch", func(t *testing.T) {
		req := handler.Request{
			ProjectId:       project.ID,
			Branch:          "",
			EnvironmentSlug: "production",
			DockerImage:     ptr.P("nginx:latest"),
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("empty environmentSlug", func(t *testing.T) {
		req := handler.Request{
			ProjectId:       project.ID,
			Branch:          "main",
			EnvironmentSlug: "",
			DockerImage:     ptr.P("nginx:latest"),
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("buildContext with missing buildContextPath", func(t *testing.T) {
		req := handler.Request{
			ProjectId:       project.ID,
			Branch:          "main",
			EnvironmentSlug: "production",
			BuildContext: &struct {
				BuildContextPath string  `json:"buildContextPath"`
				DockerfilePath   *string `json:"dockerfilePath,omitempty"`
			}{},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})
}
