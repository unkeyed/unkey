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
	Request  = openapi.V2DeployCreateDeploymentRequestBody
	Response = openapi.V2DeployCreateDeploymentResponseBody
)

type Handler struct {
	Logger     logging.Logger
	DB         db.Database
	Keys       keys.KeyService
	CtrlClient ctrlv1connect.DeploymentServiceClient
}

func (h *Handler) Path() string {
	return "/v2/deploy.createDeployment"
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
			Action:       rbac.CreateDeployment,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Project,
			ResourceID:   req.ProjectId,
			Action:       rbac.CreateDeployment,
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

	// Get docker image from request - only image deployments are supported via API
	imageSource, imageErr := req.AsV2DeployImageSource()
	if imageErr != nil || imageSource.Image == "" {
		return fault.New("docker_image is required",
			fault.Internal("failed to parse image source or empty image"),
			fault.Public("A docker_image must be provided. Build from source is only supported via GitHub integration."),
		)
	}

	// nolint: exhaustruct // optional proto fields, only setting whats provided
	ctrlReq := &ctrlv1.CreateDeploymentRequest{
		ProjectId:       req.ProjectId,
		Branch:          req.Branch,
		EnvironmentSlug: req.EnvironmentSlug,
		DockerImage:     imageSource.Image,
		GitCommit:       &ctrlv1.GitCommitInfo{},
	}

	// Add optional keyspace ID for authentication
	if req.KeyspaceId != nil {
		ctrlReq.KeyspaceId = req.KeyspaceId
	}

	// Handle optional git commit info
	if req.GitCommit != nil {
		// nolint: exhaustruct // optional proto fields, only setting whats provided
		gitCommit := &ctrlv1.GitCommitInfo{}
		if req.GitCommit.CommitSha != nil {
			gitCommit.CommitSha = *req.GitCommit.CommitSha
		}
		if req.GitCommit.CommitMessage != nil {
			gitCommit.CommitMessage = *req.GitCommit.CommitMessage
		}
		if req.GitCommit.AuthorHandle != nil {
			gitCommit.AuthorHandle = *req.GitCommit.AuthorHandle
		}
		if req.GitCommit.AuthorAvatarUrl != nil {
			gitCommit.AuthorAvatarUrl = *req.GitCommit.AuthorAvatarUrl
		}
		if req.GitCommit.Timestamp != nil {
			gitCommit.Timestamp = *req.GitCommit.Timestamp
		}
		ctrlReq.GitCommit = gitCommit
	}

	connectReq := connect.NewRequest(ctrlReq)

	ctrlResp, err := h.CtrlClient.CreateDeployment(ctx, connectReq)
	if err != nil {
		return ctrlclient.HandleError(err, "create deployment")
	}

	return s.JSON(http.StatusCreated, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2DeployCreateDeploymentResponseData{
			DeploymentId: ctrlResp.Msg.GetDeploymentId(),
		},
	})
}
