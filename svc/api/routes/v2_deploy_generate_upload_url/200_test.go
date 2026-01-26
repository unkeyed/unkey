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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_generate_upload_url"
)

func TestGenerateUploadUrlSuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger: h.Logger,
		DB:     h.DB,
		Keys:   h.Keys,
		CtrlClient: &testutil.MockDeploymentClient{
			CreateS3UploadURLFunc: func(ctx context.Context, req *connect.Request[ctrlv1.CreateS3UploadURLRequest]) (*connect.Response[ctrlv1.CreateS3UploadURLResponse], error) {
				return connect.NewResponse(&ctrlv1.CreateS3UploadURLResponse{
					UploadUrl:        "https://s3.example.com/upload",
					BuildContextPath: "s3://bucket/path/to/context.tar.gz",
				}), nil
			},
		},
	}
	h.Register(route)

	t.Run("generate upload URL successfully", func(t *testing.T) {
		setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
			SkipEnvironment: true,
			Permissions:     []string{"project.*.generate_upload_url"},
		})

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", setup.RootKey)},
		}

		req := handler.Request{
			ProjectId: setup.Project.ID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.UploadUrl, "upload URL should not be empty")
		require.NotEmpty(t, res.Body.Data.Context, "build context path should not be empty")
	})
}

func TestGenerateUploadUrlWithWildcardPermission(t *testing.T) {
	t.Parallel()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger: h.Logger,
		DB:     h.DB,
		Keys:   h.Keys,
		CtrlClient: &testutil.MockDeploymentClient{
			CreateS3UploadURLFunc: func(ctx context.Context, req *connect.Request[ctrlv1.CreateS3UploadURLRequest]) (*connect.Response[ctrlv1.CreateS3UploadURLResponse], error) {
				return connect.NewResponse(&ctrlv1.CreateS3UploadURLResponse{
					UploadUrl:        "https://s3.example.com/upload",
					BuildContextPath: "s3://bucket/path/to/context.tar.gz",
				}), nil
			},
		},
	}
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		SkipEnvironment: true,
		Permissions:     []string{"project.*.generate_upload_url"},
	})

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", setup.RootKey)},
	}

	req := handler.Request{
		ProjectId: setup.Project.ID,
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusOK, res.Status, "Expected 200, got: %d", res.Status)
	require.NotNil(t, res.Body)
}

func TestGenerateUploadUrlWithSpecificProjectPermission(t *testing.T) {
	t.Parallel()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger: h.Logger,
		DB:     h.DB,
		Keys:   h.Keys,
		CtrlClient: &testutil.MockDeploymentClient{
			CreateS3UploadURLFunc: func(ctx context.Context, req *connect.Request[ctrlv1.CreateS3UploadURLRequest]) (*connect.Response[ctrlv1.CreateS3UploadURLResponse], error) {
				return connect.NewResponse(&ctrlv1.CreateS3UploadURLResponse{
					UploadUrl:        "https://s3.example.com/upload",
					BuildContextPath: "s3://bucket/path/to/context.tar.gz",
				}), nil
			},
		},
	}
	h.Register(route)

	// First create the project setup
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		SkipEnvironment: true,
	})

	// Now create a root key with project-specific permission
	rootKey := h.CreateRootKey(setup.Workspace.ID, fmt.Sprintf("project.%s.generate_upload_url", setup.Project.ID))

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		ProjectId: setup.Project.ID,
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusOK, res.Status, "Expected 200, got: %d", res.Status)
	require.NotNil(t, res.Body)
}
