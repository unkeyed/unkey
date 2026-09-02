package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/deployfail"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V3DeploymentsCreateDeploymentRequestBody
	Response = openapi.V3DeploymentsCreateDeploymentResponseBody
)

type Handler struct {
	DB         db.Database
	CtrlClient ctrl.DeployServiceClient
}

func (h *Handler) Path() string {
	return "/v3/deployments.createDeployment"
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
		rbac.U(
			urn.New().Workspace(principal.WorkspaceID).Project(environment.ProjectID).App(environment.AppID).Environment(environment.ID).Deployment("*"),
			permissions.CreateDeployment{},
		),
	))
	if err != nil {
		return err
	}

	client := s.Request().Header.Get("X-Unkey-Client")
	trigger := ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_API
	switch {
	case strings.HasPrefix(client, "unkey-cli/"):
		trigger = ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_CLI
	case strings.HasPrefix(client, "unkey-dashboard"):
		trigger = ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_DASHBOARD
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
	case req.Oci != nil:
		ctrlReq.OciImage = req.Oci.Image

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
		if _, err = db.Query.FindGithubRepoConnectionByAppId(ctx, h.DB.RO(), environment.AppID); err != nil {
			if db.IsNotFound(err) {
				return fault.New(
					"no repo connection",
					fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
					fault.Internal("app has no github repo connection for git source"),
					fault.Public("This app has no connected GitHub repository. Deploy a prebuilt image with the OCI source, or connect a repository first."),
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
		gitCommit, ociImage, resolveErr := h.resolveRedeploy(ctx, principal.WorkspaceID, environment.AppID, environment.ID, req.Deployment.DeploymentId)
		if resolveErr != nil {
			return resolveErr
		}
		ctrlReq.GitCommit = gitCommit
		ctrlReq.OciImage = ociImage
	}

	if err = h.ensureEnvironmentDeployable(ctx, environment); err != nil {
		return err
	}

	ctrlResp, err := h.CtrlClient.CreateDeployment(ctx, ctrlReq)
	if err != nil {
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
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V3DeploymentsCreateDeploymentResponseData{DeploymentId: ctrlResp.GetDeploymentId()},
	})
}

func (h *Handler) resolveRedeploy(ctx context.Context, workspaceID, appID, environmentID, deploymentID string) (*ctrlv1.GitCommitInfo, string, error) {
	deployment, err := db.Query.FindDeploymentById(ctx, h.DB.RO(), deploymentID)
	if err != nil && !db.IsNotFound(err) {
		return nil, "", fault.Wrap(err, fault.Internal("failed to find deployment"))
	}

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

	resolvedImage := func() (*ctrlv1.GitCommitInfo, string, error) {
		image := deployment.ImageResolved
		if !image.Valid || image.String == "" {
			image = deployment.Image
		}
		if !image.Valid || image.String == "" {
			return nil, "", fault.New(
				"deployment not redeployable",
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Internal("redeploy target has no resolved image"),
				fault.Public("This deployment cannot be redeployed because it never produced an image."),
			)
		}
		return nil, image.String, nil
	}
	gitCommit := func(requireSHA bool) (*ctrlv1.GitCommitInfo, string, error) {
		if requireSHA && deployment.GitCommitSha.String == "" {
			return nil, "", fault.New(
				"deployment not redeployable",
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Internal("git redeploy target has no commit SHA"),
				fault.Public("This deployment cannot be redeployed because its Git commit is unavailable."),
			)
		}
		_, connectionErr := db.Query.FindGithubRepoConnectionByAppId(ctx, h.DB.RO(), appID)
		if db.IsNotFound(connectionErr) {
			return nil, "", fault.New(
				"deployment not redeployable",
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Internal("git redeploy target has no repository connection"),
				fault.Public("This deployment cannot be redeployed because its repository is not connected."),
			)
		}
		if connectionErr != nil {
			return nil, "", fault.Wrap(connectionErr, fault.Internal("failed to check repo connection"))
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
	}

	switch deployment.Source {
	case db.DeploymentsSourceGit:
		return gitCommit(true)
	case db.DeploymentsSourceOci:
		return resolvedImage()
	case db.DeploymentsSourceUnknown, "":
		_, connectionErr := db.Query.FindGithubRepoConnectionByAppId(ctx, h.DB.RO(), appID)
		if connectionErr == nil && (deployment.GitBranch.String != "" || deployment.GitCommitSha.String != "") {
			return gitCommit(false)
		}
		if connectionErr != nil && !db.IsNotFound(connectionErr) {
			return nil, "", fault.Wrap(connectionErr, fault.Internal("failed to check repo connection"))
		}
		return resolvedImage()
	default:
		return nil, "", fault.New(
			"deployment not redeployable",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal(fmt.Sprintf("unsupported deployment source %q", deployment.Source)),
			fault.Public("This deployment has an unsupported source and cannot be redeployed."),
		)
	}
}

func (h *Handler) ensureEnvironmentDeployable(ctx context.Context, environment db.Environment) error {
	runtime, err := db.Query.FindAppRuntimeSettingsByAppAndEnv(ctx, h.DB.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
		AppID:         environment.AppID,
		EnvironmentID: environment.ID,
	})
	if err != nil && !db.IsNotFound(err) {
		return fault.Wrap(err, fault.Internal("failed to load runtime settings"))
	}

	var problems []string
	if db.IsNotFound(err) {
		problems = append(problems, "runtime settings are not configured")
	} else {
		for _, violation := range deployfail.RuntimeViolations(runtime.Port, runtime.CpuMillicores, runtime.MemoryMib) {
			problems = append(problems, fmt.Sprintf("%s (is %d)", violation.Message, violation.Actual))
		}
	}

	regional, err := db.Query.FindAppRegionalSettingsByAppAndEnv(ctx, h.DB.RO(), db.FindAppRegionalSettingsByAppAndEnvParams{
		AppID:         environment.AppID,
		EnvironmentID: environment.ID,
	})
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to load regional settings"))
	}
	if !slices.ContainsFunc(regional, func(region db.FindAppRegionalSettingsByAppAndEnvRow) bool { return region.RegionCanSchedule }) {
		problems = append(problems, "no schedulable regions are configured")
	}

	if len(problems) == 0 {
		return nil
	}

	joined := strings.Join(problems, "; ")
	return fault.New(
		"environment not deployable",
		fault.Code(codes.App.Validation.InvalidEnvironmentSettings.URN()),
		fault.Internal(fmt.Sprintf("environment %s fails deploy preconditions: %s", environment.Slug, joined)),
		fault.Public(fmt.Sprintf("Environment %q cannot be deployed: %s. Update the environment's settings before deploying.", environment.Slug, joined)),
	)
}

func hasValue(value *string) bool {
	return value != nil && strings.TrimSpace(*value) != ""
}
