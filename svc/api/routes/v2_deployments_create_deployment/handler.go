package handler

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	restate "github.com/restatedev/sdk-go"
	restateingress "github.com/restatedev/sdk-go/ingress"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/deployfail"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/internal/deployment"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2DeploymentsCreateDeploymentRequestBody
	Response = openapi.V2DeploymentsCreateDeploymentResponseBody
)

type Handler struct {
	DB      db.Database
	Restate *restateingress.Client
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
		rbac.U(
			urn.New().Workspace(principal.WorkspaceID).Project(environment.ProjectID).App(environment.AppID).Environment(environment.ID).Deployment("*"),
			permissions.CreateDeployment{},
		),
	))
	if err != nil {
		return err
	}

	// Clients announce themselves via X-Unkey-Client: the CLI as
	// unkey-cli/<version>, the dashboard proxy as unkey-dashboard. Anything else
	// (or absent) is attributed to the API.
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

	// The id is minted here because it is the Restate object key the create
	// runs on, so the response can name the deployment without waiting.
	//
	// With an Idempotency-Key the id is derived from it instead of random, so a
	// retry lands on the same object and finds the row the first attempt wrote.
	// The key also goes to Restate, which replays the first response inside its
	// retention window rather than running the create again. Without the header
	// a retry is simply a second deployment, which is what a caller that did
	// not ask for idempotency gets.
	idempotencyKey := s.Request().Header.Get(deployment.IdempotencyKeyHeader)
	if err = deployment.ValidateIdempotencyKey(idempotencyKey); err != nil {
		return err
	}

	deploymentID := uid.New(uid.DeploymentPrefix)
	var sendOpts []restate.IngressSendOption
	if idempotencyKey != "" {
		// The tenant and the target are in the scope, so one key cannot derive
		// the same id for two apps. That is what lets Create refuse a row
		// belonging to another target instead of replaying it as this caller's.
		deploymentID = uid.Derived(uid.DeploymentPrefix,
			principal.WorkspaceID, environment.AppID, environment.ID, idempotencyKey)
		sendOpts = append(sendOpts, restate.WithIdempotencyKey(idempotencyKey))
	}

	// nolint: exhaustruct // the source oneof is set per case below
	createReq := &hydrav1.DeployCreateRequest{
		ProjectId:   environment.ProjectID,
		AppId:       environment.AppID,
		Environment: environment.ID,
		Decision:    hydrav1.CreateDecision_CREATE_DECISION_DEPLOY,
		Command:     nil,
		// Zero lets the worker stamp created_at from its own clock. Only the
		// GitHub webhook needs to pin an ordering timestamp, to keep push order
		// across its per-app sends.
		OrderingTimestamp: 0,
		Trigger:           trigger,
		TriggeredBy:       principal.Subject.ID,
		TriggerReason:     "",
		Actor:             actorInfo,
	}

	switch {
	case req.Image != nil:
		createReq.Source = &hydrav1.DeployCreateRequest_Image{
			Image: &hydrav1.CreateImageSource{Image: req.Image.DockerImage},
		}

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
		createReq.Source = &hydrav1.DeployCreateRequest_Git{
			// nolint: exhaustruct // the worker fills the commit metadata it resolves from git
			Git: &hydrav1.CreateGitSource{
				Commit: &ctrlv1.GitCommitInfo{
					Branch:         ptr.SafeDeref(git.Branch),
					CommitSha:      ptr.SafeDeref(git.CommitSha),
					ForkRepository: ptr.SafeDeref(git.Repository),
				},
				PrNumber: 0,
			},
		}

	case req.Deployment != nil:
		gitCommit, dockerImage, err := h.resolveRedeploy(ctx, principal.WorkspaceID, environment.AppID, environment.ID, req.Deployment.DeploymentId)
		if err != nil {
			return err
		}
		// resolveRedeploy returns the source deployment's commit when the app is
		// still git-connected, otherwise its image, so exactly one of the two is
		// set and the redeploy runs as a plain git or image create.
		if gitCommit != nil {
			createReq.Source = &hydrav1.DeployCreateRequest_Git{
				Git: &hydrav1.CreateGitSource{Commit: gitCommit, PrNumber: 0},
			}
		} else {
			createReq.Source = &hydrav1.DeployCreateRequest_Image{
				Image: &hydrav1.CreateImageSource{Image: dockerImage},
			}
		}

	default:
		return fault.New(
			"exactly one source required",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("no source set after validation"),
			fault.Public("Provide exactly one of image, git, or deployment."),
		)
	}

	// Reject environments whose runtime/regional settings would fail the deploy
	// pipeline before enqueuing, so callers get a synchronous 400 instead of a
	// build that dies mid-flight. The worker keeps the same asserts as a backstop.
	if err = h.ensureEnvironmentDeployable(ctx, environment); err != nil {
		return err
	}

	// The create is submitted one-way below, so this is the only place a caller
	// can be told its workspace may not deploy.
	if err := deployment.EnsureWorkspaceCanDeploy(ctx, h.DB, principal.WorkspaceID); err != nil {
		return err
	}

	// Send, not Request: everything the caller has to see is settled above, and
	// the create itself resolves commits against GitHub and retries transient
	// failures for minutes, which no HTTP caller should wait through. The worker
	// re-checks these gates when it runs, so a workspace that loses its
	// entitlement between here and there is still refused, just asynchronously.
	if _, err := hydrav1.NewDeployServiceIngressClient(h.Restate, deploymentID).
		Create().
		Send(ctx, createReq, sendOpts...); err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to submit deployment create to Restate"),
			fault.Public("Failed to create deployment."),
		)
	}

	return s.JSON(http.StatusCreated, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2DeploymentsCreateDeploymentResponseData{
			DeploymentId: deploymentID,
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

// ensureEnvironmentDeployable mirrors the deploy worker's runtime and region
// preconditions (svc/ctrl/worker/deploy: port 1..65535, cpu>0, mem>0, and at
// least one schedulable region) so an undeployable environment is rejected up
// front rather than after a full build. It reports every failing field in one
// message so the caller can fix them in a single pass.
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
		s := runtime.AppRuntimeSetting
		for _, v := range deployfail.RuntimeViolations(s.Port, s.CpuMillicores, s.MemoryMib) {
			problems = append(problems, fmt.Sprintf("%s (is %d)", v.Message, v.Actual))
		}
	}

	regional, err := db.Query.FindAppRegionalSettingsByAppAndEnv(ctx, h.DB.RO(), db.FindAppRegionalSettingsByAppAndEnvParams{
		AppID:         environment.AppID,
		EnvironmentID: environment.ID,
	})
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to load regional settings"))
	}
	if !slices.ContainsFunc(regional, func(r db.FindAppRegionalSettingsByAppAndEnvRow) bool { return r.RegionCanSchedule }) {
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

func hasValue(p *string) bool {
	return p != nil && strings.TrimSpace(*p) != ""
}
