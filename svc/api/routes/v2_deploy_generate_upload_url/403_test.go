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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_generate_upload_url"
)

func TestGenerateUploadUrlInsufficientPermissions(t *testing.T) {
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

	// Create setup with insufficient permissions
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"project.*.create_deployment"},
	})

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", setup.RootKey)},
	}

	req := handler.Request{
		ProjectId: setup.Project.ID,
	}

	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
	require.Equal(t, http.StatusForbidden, res.Status)
	require.NotNil(t, res.Body)
}
