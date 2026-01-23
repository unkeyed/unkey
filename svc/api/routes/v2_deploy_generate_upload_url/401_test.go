package handler_test

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_generate_upload_url"
)

func TestUnauthorizedAccess(t *testing.T) {
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

	t.Run("invalid authorization token", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer invalid_token"},
		}

		req := handler.Request{
			ProjectId: setup.Project.ID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusUnauthorized, res.Status, "expected 401, received: %s", res.RawBody)
		require.NotNil(t, res.Body)
	})
}
