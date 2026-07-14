package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2DeploymentsCreateDeploymentRequestBody
	Response = openapi.V2DeploymentsCreateDeploymentResponseBody
)

type Handler struct {
	DB         db.Database
	CtrlClient ctrl.DeployServiceClient
}

func (h *Handler) Path() string {
	return "/v2/deployments.createDeployment"
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	environment, err := db.Query.FindEnvironmentByIdentifiers(ctx, h.DB.RO(), db.FindEnvironmentByIdentifiersParams{
		WorkspaceID: principal.WorkspaceID,
		Project:     req.Project,
		App:         req.App,
		Environment: req.Environment,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New(
				"environment not found",
				fault.Code(codes.Data.Environment.NotFound.URN()),
				fault.Internal("project, app, or environment did not resolve"),
				fault.Public("The requested project, app, or environment does not exist."),
			)
		}
		return fault.Wrap(err, fault.Internal("failed to resolve environment"))
	}

	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   "*",
			Action:       rbac.CreateDeployment,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   environment.ID,
			Action:       rbac.CreateDeployment,
		}),
	))
	if err != nil {
		return err
	}

	// CLI announces itself via X-Unkey-Client: unkey-cli/<version>.
	// Anything else (or absent) is attributed to the API.
	trigger := ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_API
	if strings.HasPrefix(s.Request().Header.Get("X-Unkey-Client"), "unkey-cli/") {
		trigger = ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_CLI
	}

	actorInfo, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}

	// nolint: exhaustruct // optional proto fields are set per source below
	ctrlReq := &ctrlv1.CreateDeploymentRequest{
		ProjectId:       environment.ProjectID,
		AppId:           environment.AppID,
		EnvironmentSlug: environment.Slug,
		Trigger:         trigger,
		TriggeredBy:     principal.Subject.ID,
		Actor:           actorInfo,
	}

	switch {
	case req.Image != nil:
		ctrlReq.DockerImage = req.Image.DockerImage

	case req.Git != nil:
		git := req.Git
		if hasValue(git.Repository) && !hasValue(git.CommitSha) {
			return fault.New(
				"repository requires commitSha",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("repository set without commitSha"),
				fault.Public("repository requires commitSha."),
			)
		}
		// Git builds need a connected repository. ctrl resolves branch/commit
		// against the app's installation, so a missing connection is a caller
		// precondition, not an internal failure.
		if _, err = db.Query.FindGithubRepoConnectionByAppId(ctx, h.DB.RO(), environment.AppID); err != nil {
			if db.IsNotFound(err) {
				return fault.New(
					"no repo connection",
					fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
					fault.Internal("app has no github repo connection for git source"),
					fault.Public("This app has no connected GitHub repository. Deploy a prebuilt image with the image source, or connect a repository first."),
				)
			}
			return fault.Wrap(err, fault.Internal("failed to check repo connection"))
		}
		// nolint: exhaustruct // ctrl fills the commit metadata it resolves from git
		ctrlReq.GitCommit = &ctrlv1.GitCommitInfo{
			Branch:         ptr.SafeDeref(git.Branch),
			CommitSha:      ptr.SafeDeref(git.CommitSha),
			ForkRepository: ptr.SafeDeref(git.Repository),
		}

	case req.Deployment != nil:
		gitCommit, dockerImage, err := h.resolveRedeploy(ctx, principal.WorkspaceID, environment.AppID, environment.ID, req.Deployment.DeploymentId)
		if err != nil {
			return err
		}
		ctrlReq.GitCommit = gitCommit
		ctrlReq.DockerImage = dockerImage

	default:
		return fault.New(
			"exactly one source required",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("no source set after validation"),
			fault.Public("Provide exactly one of image, git, or deployment."),
		)
	}

	ctrlResp, err := h.CtrlClient.CreateDeployment(ctx, ctrlReq)
	if err != nil {
		// Map ctrl's precondition failure to a 412 instead of a 500. Keep its
		// message in the internal error but return a fixed public reason so callers can't probe upstream state.
		var connectErr *connect.Error
		if errors.As(err, &connectErr) && connectErr.Code() == connect.CodeFailedPrecondition {
			return fault.Wrap(
				err,
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Internal("ctrl reported a precondition failure: "+connectErr.Message()),
				fault.Public("The deployment could not be started because a precondition was not met. Verify the app's repository connection, branch, commit, and current deployment, then try again."),
			)
		}
		return ctrlclient.HandleError(err, "create deployment")
	}

	return s.JSON(http.StatusCreated, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2DeploymentsCreateDeploymentResponseData{
			DeploymentId: ctrlResp.GetDeploymentId(),
		},
	})
}

func (h *Handler) resolveRedeploy(ctx context.Context, workspaceID, appID, environmentID, deploymentID string) (*ctrlv1.GitCommitInfo, string, error) {
	deployment, err := db.Query.FindDeploymentById(ctx, h.DB.RO(), deploymentID)
	if err != nil && !db.IsNotFound(err) {
		return nil, "", fault.Wrap(err, fault.Internal("failed to find deployment"))
	}

	// The deployment must match the exact workspace, app, and environment being
	// deployed to. Anything else - not found, another workspace, or another
	// app/environment the caller may not even have access to - is masked as not
	// found so the endpoint cannot probe for a deployment's existence.
	if db.IsNotFound(err) ||
		deployment.WorkspaceID != workspaceID ||
		deployment.AppID != appID ||
		deployment.EnvironmentID != environmentID {
		return nil, "", fault.New(
			"deployment not found",
			fault.Code(codes.Data.Deployment.NotFound.URN()),
			fault.Internal("deployment does not exist or does not match this workspace, app, and environment"),
			fault.Public("The specified deployment does not exist."),
		)
	}

	_, err = db.Query.FindGithubRepoConnectionByAppId(ctx, h.DB.RO(), appID)
	switch {
	case err == nil:
		if deployment.GitBranch.String == "" && deployment.GitCommitSha.String == "" {
			if deployment.Image.String == "" {
				return nil, "", fault.New(
					"deployment not redeployable",
					fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
					fault.Internal("redeploy target has neither git metadata nor image"),
					fault.Public("This deployment cannot be redeployed because it never produced an image."),
				)
			}
			return nil, deployment.Image.String, nil
		}
		return &ctrlv1.GitCommitInfo{
			CommitSha:       deployment.GitCommitSha.String,
			Branch:          deployment.GitBranch.String,
			CommitMessage:   deployment.GitCommitMessage.String,
			AuthorHandle:    deployment.GitCommitAuthorHandle.String,
			AuthorAvatarUrl: deployment.GitCommitAuthorAvatarUrl.String,
			Timestamp:       deployment.GitCommitTimestamp.Int64,
			ForkRepository:  deployment.ForkRepositoryFullName.String,
		}, "", nil
	case db.IsNotFound(err):
		return nil, deployment.Image.String, nil
	default:
		return nil, "", fault.Wrap(err, fault.Internal("failed to check repo connection"))
	}
}

func hasValue(p *string) bool {
	return p != nil && strings.TrimSpace(*p) != ""
}
