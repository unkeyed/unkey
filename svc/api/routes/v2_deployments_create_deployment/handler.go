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

	if err = validateSourceFields(req); err != nil {
		return err
	}

	environment, err := db.Query.FindEnvironmentByIdentifiers(ctx, h.DB.RO(), db.FindEnvironmentByIdentifiersParams{
		WorkspaceID: principal.WorkspaceID,
		Project:     req.Project,
		App:         req.App,
		Environment: req.EnvironmentSlug,
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

	switch req.Source {
	case openapi.DeploymentSourceImage:
		ctrlReq.DockerImage = *req.DockerImage

	case openapi.DeploymentSourceGit:
		// Git builds need a connected repository. ctrl resolves branch/commit
		// against the app's installation, so a missing connection is a caller
		// precondition, not an internal failure.
		if _, err = db.Query.FindGithubRepoConnectionByAppId(ctx, h.DB.RO(), environment.AppID); err != nil {
			if db.IsNotFound(err) {
				return fault.New(
					"no repo connection",
					fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
					fault.Internal("app has no github repo connection for git source"),
					fault.Public("This app has no connected GitHub repository. Deploy a prebuilt image with source=image, or connect a repository first."),
				)
			}
			return fault.Wrap(err, fault.Internal("failed to check repo connection"))
		}
		// nolint: exhaustruct // ctrl fills the commit metadata it resolves from git
		ctrlReq.GitCommit = &ctrlv1.GitCommitInfo{
			Branch:         ptr.SafeDeref(req.Branch),
			CommitSha:      ptr.SafeDeref(req.CommitSha),
			ForkRepository: ptr.SafeDeref(req.ForkRepository),
		}

	case openapi.DeploymentSourceDeployment:
		gitCommit, dockerImage, err := h.resolveRedeploy(ctx, principal.WorkspaceID, environment.AppID, environment.ID, *req.DeploymentId)
		if err != nil {
			return err
		}
		ctrlReq.GitCommit = gitCommit
		ctrlReq.DockerImage = dockerImage

	default:
		return fault.New(
			"unknown source",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("unknown source reached switch after validation"),
			fault.Public("Unknown source. Use image, git, or deployment."),
		)
	}

	ctrlResp, err := h.CtrlClient.CreateDeployment(ctx, ctrlReq)
	if err != nil {
		// ctrl may report a precondition failure; HandleError would turn it into
		// a 500, so surface a 412 instead. Don't forward ctrl's message: it can
		// carry upstream detail and lets callers probe internal state. Log it,
		// return a fixed reason.
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

func validateSourceFields(req Request) error {
	switch req.Source {
	case openapi.DeploymentSourceImage:
		if !hasValue(req.DockerImage) {
			return fault.New(
				"missing dockerImage",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("dockerImage missing for image source"),
				fault.Public("A dockerImage is required when source is image."),
			)
		}

	case openapi.DeploymentSourceGit:
		if hasValue(req.ForkRepository) && !hasValue(req.CommitSha) {
			return fault.New(
				"forkRepository requires commitSha",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("forkRepository set without commitSha"),
				fault.Public("forkRepository requires commitSha."),
			)
		}

	case openapi.DeploymentSourceDeployment:
		if !hasValue(req.DeploymentId) {
			return fault.New(
				"missing deploymentId",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("deploymentId missing for deployment source"),
				fault.Public("A deploymentId is required when source is deployment."),
			)
		}

	default:
		return fault.New(
			"unknown source",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("unknown source value"),
			fault.Public("Unknown source. Use image, git, or deployment."),
		)
	}

	return nil
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
