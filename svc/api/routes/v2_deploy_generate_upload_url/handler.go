package handler

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2DeployGenerateUploadUrlRequestBody
	Response = openapi.V2DeployGenerateUploadUrlResponseBody
)

type Handler struct {
	Logger     logging.Logger
	DB         db.Database
	Keys       keys.KeyService
	CtrlClient ctrlv1connect.BuildServiceClient
}

func (h *Handler) Path() string {
	return "/v2/deploy.generateUploadUrl"
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Project,
			ResourceID:   "*",
			Action:       rbac.GenerateUploadURL,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Project,
			ResourceID:   req.ProjectId,
			Action:       rbac.GenerateUploadURL,
		}),
	)))
	if err != nil {
		return err
	}

	// Verify project belongs to the authenticated workspace
	project, err := db.Query.FindProjectById(ctx, h.DB.RO(), req.ProjectId)
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("project not found",
				fault.Code(codes.Data.Project.NotFound.URN()),
				fault.Internal("project not found"),
				fault.Public("The requested project does not exist or has been deleted."),
			)
		}
		return fault.Wrap(err, fault.Internal("failed to find project"))
	}
	if project.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("wrong workspace",
			fault.Code(codes.Data.Project.NotFound.URN()),
			fault.Internal("wrong workspace, masking as 404"),
			fault.Public("The requested project does not exist or has been deleted."),
		)
	}

	ctrlReq := &ctrlv1.GenerateUploadURLRequest{
		UnkeyProjectId: req.ProjectId,
	}

	connectReq := connect.NewRequest(ctrlReq)

	ctrlResp, err := h.CtrlClient.GenerateUploadURL(ctx, connectReq)
	if err != nil {
		return ctrlclient.HandleError(err, "generate upload URL")
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2DeployGenerateUploadUrlResponseData{
			UploadUrl: ctrlResp.Msg.GetUploadUrl(),
			Context:   ctrlResp.Msg.GetBuildContextPath(),
		},
	})
}
