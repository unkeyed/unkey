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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_create_deployment"
)

func TestBadRequests(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger: h.Logger,
		DB:     h.DB,
		Keys:   h.Keys,
		CtrlClient: &testutil.MockDeploymentClient{
			CreateDeploymentFunc: func(ctx context.Context, req *connect.Request[ctrlv1.CreateDeploymentRequest]) (*connect.Response[ctrlv1.CreateDeploymentResponse], error) {
				return connect.NewResponse(&ctrlv1.CreateDeploymentResponse{DeploymentId: "test-deployment-id"}), nil
			},
		},
	}
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"project.*.create_deployment"},
	})

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", setup.RootKey)},
	}

	t.Run("missing both build and image", func(t *testing.T) {
		req := handler.Request{
			ProjectId:       setup.Project.ID,
			Branch:          "main",
			EnvironmentSlug: "production",
			// Neither build nor image provided
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		// The OpenAPI schema validator catches this with a generic schema validation error
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("both build and image provided", func(t *testing.T) {
		req := handler.Request{
			ProjectId:       setup.Project.ID,
			Branch:          "main",
			EnvironmentSlug: "production",
		}

		// Manually set both build and image in the union by merging both types
		// This tests that the OpenAPI oneOf validation rejects requests with both sources
		_ = req.FromV2DeployBuildSource(openapi.V2DeployBuildSource{
			Build: struct {
				Context    string  "json:\"context\""
				Dockerfile *string "json:\"dockerfile,omitempty\""
			}{
				Context: "/app",
			},
		})
		_ = req.MergeV2DeployImageSource(openapi.V2DeployImageSource{
			Image: "nginx:latest",
		})

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400 when both build and image are provided, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		// The OpenAPI schema validator catches this with a generic schema validation error
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("missing projectId", func(t *testing.T) {
		req := handler.Request{
			Branch:          "main",
			EnvironmentSlug: "production",
		}
		_ = req.FromV2DeployImageSource(openapi.V2DeployImageSource{
			Image: "nginx:latest",
		})

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
	})

	t.Run("missing branch", func(t *testing.T) {
		req := handler.Request{
			ProjectId:       setup.Project.ID,
			EnvironmentSlug: "production",
		}
		_ = req.FromV2DeployImageSource(openapi.V2DeployImageSource{
			Image: "nginx:latest",
		})

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
			ProjectId: setup.Project.ID,
			Branch:    "main",
		}

		err := req.FromV2DeployImageSource(openapi.V2DeployImageSource{
			Image: "nginx:latest",
		})
		require.NoError(t, err, "failed to set image source")

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
			ProjectId:       setup.Project.ID,
			Branch:          "main",
			EnvironmentSlug: "production",
		}

		err := req.FromV2DeployImageSource(openapi.V2DeployImageSource{
			Image: "nginx:latest",
		})
		require.NoError(t, err, "failed to set image source")

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
			ProjectId:       setup.Project.ID,
			Branch:          "main",
			EnvironmentSlug: "production",
		}

		err := req.FromV2DeployImageSource(openapi.V2DeployImageSource{
			Image: "nginx:latest",
		})
		require.NoError(t, err, "failed to set image source")

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("empty projectId", func(t *testing.T) {
		req := handler.Request{
			ProjectId:       "",
			Branch:          "main",
			EnvironmentSlug: "production",
		}

		err := req.FromV2DeployImageSource(openapi.V2DeployImageSource{
			Image: "nginx:latest",
		})
		require.NoError(t, err, "failed to set image source")

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("empty branch", func(t *testing.T) {
		req := handler.Request{
			ProjectId:       setup.Project.ID,
			Branch:          "",
			EnvironmentSlug: "production",
		}

		err := req.FromV2DeployImageSource(openapi.V2DeployImageSource{
			Image: "nginx:latest",
		})
		require.NoError(t, err, "failed to set image source")

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("empty environmentSlug", func(t *testing.T) {
		req := handler.Request{
			ProjectId:       setup.Project.ID,
			Branch:          "main",
			EnvironmentSlug: "",
		}

		err := req.FromV2DeployImageSource(openapi.V2DeployImageSource{
			Image: "nginx:latest",
		})
		require.NoError(t, err, "failed to set image source")

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("build with missing context", func(t *testing.T) {
		req := handler.Request{
			ProjectId:       setup.Project.ID,
			Branch:          "main",
			EnvironmentSlug: "production",
		}

		err := req.FromV2DeployBuildSource(openapi.V2DeployBuildSource{
			Build: struct {
				Context    string  "json:\"context\""
				Dockerfile *string "json:\"dockerfile,omitempty\""
			}{},
		})
		require.NoError(t, err, "failed to set build source")

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})
}
