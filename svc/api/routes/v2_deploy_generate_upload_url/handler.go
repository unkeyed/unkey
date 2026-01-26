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
	// Request is the request body for generating an upload URL, containing the
	// target project ID. Aliased from [openapi.V2DeployGenerateUploadUrlRequestBody].
	Request = openapi.V2DeployGenerateUploadUrlRequestBody

	// Response is the response body containing the pre-signed upload URL and
	// build context path. Aliased from [openapi.V2DeployGenerateUploadUrlResponseBody].
	Response = openapi.V2DeployGenerateUploadUrlResponseBody
)

// Handler generates pre-signed S3 upload URLs for deployment build contexts.
// It validates authentication, checks RBAC permissions, verifies project ownership,
// and delegates URL generation to the control plane service.
type Handler struct {
	Logger     logging.Logger
	DB         db.Database
	Keys       keys.KeyService
	CtrlClient ctrlv1connect.DeploymentServiceClient
}

// Path returns the URL path for this endpoint.
func (h *Handler) Path() string {
	return "/v2/deploy.generateUploadUrl"
}

// Method returns the HTTP method for this endpoint.
func (h *Handler) Method() string {
	return "POST"
}

// Handle processes a request to generate a pre-signed S3 upload URL. It
// authenticates via root key, verifies the caller has generate_upload_url
// permission on the project, confirms the project belongs to the caller's
// workspace, then returns an upload URL from the control plane. Returns 400
// for invalid input, 401 for invalid root key, 403 for missing permissions,
// or 404 if the project does not exist in the caller's workspace.
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

	ctrlResp, err := h.CtrlClient.CreateS3UploadURL(ctx, connect.NewRequest(&ctrlv1.CreateS3UploadURLRequest{
		UnkeyProjectId: req.ProjectId,
	}))
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
