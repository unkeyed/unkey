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
	_, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	if req.ProjectId == "" {
		return fault.New("projectId is required",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("projectId is required."),
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

	// Handle source (build vs image) using oneOf union type
	buildSource, buildErr := req.AsV2DeployBuildSource()

	if buildErr == nil && buildSource.Build.Context != "" {
		// Build source
		// nolint: exhaustruct // optional proto fields, only setting whats provided
		buildContext := &ctrlv1.BuildContext{
			BuildContextPath: buildSource.Build.Context,
		}
		if buildSource.Build.Dockerfile != nil {
			buildContext.DockerfilePath = buildSource.Build.Dockerfile
		}
		ctrlReq.Source = &ctrlv1.CreateDeploymentRequest_BuildContext{
			BuildContext: buildContext,
		}
	} else {
		// Image source
		imageSource, _ := req.AsV2DeployImageSource()
		ctrlReq.Source = &ctrlv1.CreateDeploymentRequest_DockerImage{
			DockerImage: imageSource.Image,
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
