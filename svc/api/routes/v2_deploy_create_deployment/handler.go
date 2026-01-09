package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/zen"
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
	CtrlToken  string
}

func (h *Handler) Path() string {
	return "/v2/deploy.createDeployment"
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	_, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	// TODO: We'll add RBAC permission in the following PRs when we add project permissions

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	// Validate that either buildContext or dockerImage is provided
	if req.BuildContext == nil && req.DockerImage == nil {
		return fault.New("must provide either buildContext or dockerImage",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Either buildContext or dockerImage must be provided."),
		)
	}
	if req.BuildContext != nil && req.DockerImage != nil {
		return fault.New("cannot provide both buildContext and dockerImage",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Only one of buildContext or dockerImage can be provided."),
		)
	}

	// nolint: exhaustruct // optional proto fields, only setting whats provided
	ctrlReq := &ctrlv1.CreateDeploymentRequest{
		ProjectId:       req.ProjectId,
		Branch:          req.Branch,
		EnvironmentSlug: req.EnvironmentSlug,
		GitCommit:       &ctrlv1.GitCommitInfo{},
	}

	// Add optional keyspace ID for authentication
	if req.KeyspaceId != nil {
		ctrlReq.KeyspaceId = req.KeyspaceId
	}

	// Handle source (build_context vs docker_image)
	if req.BuildContext != nil {
		// nolint: exhaustruct // optional proto fields, only setting whats provided
		buildContext := &ctrlv1.BuildContext{
			BuildContextPath: *req.BuildContext.BuildContextPath,
		}
		if req.BuildContext.DockerfilePath != nil {
			buildContext.DockerfilePath = req.BuildContext.DockerfilePath
		}
		ctrlReq.Source = &ctrlv1.CreateDeploymentRequest_BuildContext{
			BuildContext: buildContext,
		}
	} else if req.DockerImage != nil {
		ctrlReq.Source = &ctrlv1.CreateDeploymentRequest_DockerImage{
			DockerImage: *req.DockerImage,
		}
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
	connectReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", h.CtrlToken))

	ctrlResp, err := h.CtrlClient.CreateDeployment(ctx, connectReq)
	if err != nil {
		return h.handleCtrlError(err)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2DeployCreateDeploymentResponseData{
			DeploymentId: ctrlResp.Msg.GetDeploymentId(),
		},
	})
}

func (h *Handler) handleCtrlError(err error) error {
	// Convert Connect errors to fault errors
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		//nolint:exhaustive // Default case handles all other Connect error codes
		switch connectErr.Code() {
		case connect.CodeNotFound:
			return fault.Wrap(err,
				fault.Code(codes.Data.Project.NotFound.URN()),
				fault.Public("Project or environment not found."),
			)
		case connect.CodeInvalidArgument:
			return fault.Wrap(err,
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Public(connectErr.Message()),
			)
		case connect.CodeUnauthenticated:
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Public("Failed to authenticate with deployment service."),
			)
		default:
			// All other Connect errors (Internal, Unavailable, etc.)
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Public("Failed to create deployment."),
			)
		}
	}

	// Non-Connect errors
	return fault.Wrap(err,
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Public("Failed to create deployment."),
	)
}
