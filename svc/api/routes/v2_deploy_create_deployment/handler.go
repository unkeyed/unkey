package handler

import (
	"context"
	"net/http"
	"strings"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
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
	DB         db.Database
	Keys       keys.KeyService
	CtrlClient ctrl.DeployServiceClient
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

	// Resolve project + app in a single query by workspace + slugs
	row, err := db.Query.FindAppByWorkspaceAndSlugs(ctx, h.DB.RO(), db.FindAppByWorkspaceAndSlugsParams{
		WorkspaceID: auth.AuthorizedWorkspaceID,
		ProjectSlug: req.Project,
		AppSlug:     req.App,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("project or app not found",
				fault.Code(codes.Data.Project.NotFound.URN()),
				fault.Internal("project or app not found"),
				fault.Public("The requested project or app does not exist."),
			)
		}
		return fault.Wrap(err, fault.Internal("failed to find project and app"))
	}

	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Project,
			ResourceID:   "*",
			Action:       rbac.CreateDeployment,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Project,
			ResourceID:   row.Project.ID,
			Action:       rbac.CreateDeployment,
		}),
	)))
	if err != nil {
		return err
	}

	// CLI announces itself via X-Unkey-Client: unkey-cli/<version>.
	// Anything else (or absent) is attributed to the API.
	trigger := ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_API
	if strings.HasPrefix(s.Request().Header.Get("X-Unkey-Client"), "unkey-cli/") {
		trigger = ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_CLI
	}

	// nolint: exhaustruct // optional proto fields, only setting whats provided
	ctrlReq := &ctrlv1.CreateDeploymentRequest{
		ProjectId:       row.Project.ID,
		AppId:           row.App.ID,
		EnvironmentSlug: req.EnvironmentSlug,
		DockerImage:     req.DockerImage,
		GitCommit: &ctrlv1.GitCommitInfo{
			Branch: req.Branch,
		},
		Trigger:     trigger,
		TriggeredBy: auth.Key.ID,
	}

	// Add optional keyspace ID for authentication
	if req.KeyspaceId != nil {
		ctrlReq.KeyspaceId = req.KeyspaceId
	}

	// Handle optional git commit info
	if req.GitCommit != nil {
		// nolint: exhaustruct // optional proto fields, only setting whats provided
		gitCommit := &ctrlv1.GitCommitInfo{
			Branch: req.Branch,
		}
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

	ctrlResp, err := h.CtrlClient.CreateDeployment(ctx, ctrlReq)
	if err != nil {
		return ctrlclient.HandleError(err, "create deployment")
	}

	return s.JSON(http.StatusCreated, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2DeployCreateDeploymentResponseData{
			DeploymentId: ctrlResp.GetDeploymentId(),
		},
	})
}
