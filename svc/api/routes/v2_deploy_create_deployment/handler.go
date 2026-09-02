package handler

import (
	"context"
	"net/http"
	"strings"

	restate "github.com/restatedev/sdk-go"
	restateingress "github.com/restatedev/sdk-go/ingress"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/imageref"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/deployment"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2DeployCreateDeploymentRequestBody
	Response = openapi.V2DeployCreateDeploymentResponseBody
)

type Handler struct {
	DB      db.Database
	Restate *restateingress.Client
}

func (h *Handler) Path() string {
	return "/v2/deploy.createDeployment"
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

	// Resolve project + app in a single query by workspace + slugs
	row, err := db.Query.FindAppByWorkspaceAndSlugs(ctx, h.DB.RO(), db.FindAppByWorkspaceAndSlugsParams{
		WorkspaceID: principal.WorkspaceID,
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

	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Project,
			ResourceID:   "*",
			Action:       rbac.CreateDeployment,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Project,
			ResourceID:   row.ProjectID,
			Action:       rbac.CreateDeployment,
		}),
	))
	if err != nil {
		return err
	}

	// Resolving here makes an unknown slug a 404. The worker would only refuse
	// it as a blocked create, long after this response was sent.
	environment, err := db.Query.FindEnvironmentByIdentifiers(ctx, h.DB.RO(), db.FindEnvironmentByIdentifiersParams{
		WorkspaceID: principal.WorkspaceID,
		Project:     req.Project,
		App:         req.App,
		Environment: req.EnvironmentSlug,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("environment not found",
				fault.Code(codes.Data.Environment.NotFound.URN()),
				fault.Internal("environment slug did not resolve for this project and app"),
				fault.Public("The requested environment does not exist."),
			)
		}
		return fault.Wrap(err, fault.Internal("failed to resolve environment"))
	}

	// CLI announces itself via X-Unkey-Client: unkey-cli/<version>.
	// Anything else (or absent) is attributed to the API.
	trigger := ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_API
	if strings.HasPrefix(s.Request().Header.Get("X-Unkey-Client"), "unkey-cli/") {
		trigger = ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_CLI
	}

	// The create is submitted one-way, so a malformed reference has to be caught
	// here: downstream it would only surface as a failed pull.
	if err := imageref.Validate(req.DockerImage); err != nil {
		return err
	}

	// The id is the Restate object key the create runs on, so minting it here
	// lets the response name the deployment without waiting. An Idempotency-Key
	// derives that id rather than randomizing it, so a retry addresses the same
	// object and finds the row the first attempt wrote. Restate also replays the
	// first response for a key inside its retention window.
	idempotencyKey := s.Request().Header.Get(deployment.IdempotencyKeyHeader)
	if err = deployment.ValidateIdempotencyKey(idempotencyKey); err != nil {
		return err
	}

	deploymentID := uid.New(uid.DeploymentPrefix)
	var sendOpts []restate.IngressSendOption
	if idempotencyKey != "" {
		deploymentID = uid.Derived(uid.DeploymentPrefix,
			principal.WorkspaceID, environment.ID, idempotencyKey)
		sendOpts = append(sendOpts, restate.WithIdempotencyKey(idempotencyKey))
	}

	// nolint: exhaustruct // the source oneof is set below
	createReq := &hydrav1.DeployCreateRequest{
		ProjectId:      row.ProjectID,
		AppId:          row.AppID,
		Environment:    environment.ID,
		Decision:       hydrav1.CreateDecision_CREATE_DECISION_DEPLOY,
		Command:        nil,
		PushReceivedAt: 0,
		Trigger:        trigger,
		TriggeredBy:    principal.Subject.ID,
		TriggerReason:  "",
		Actor:          nil,
	}

	// Nothing downstream consumes the keyspace id, so this check only decides
	// whether to refuse the request: a root key must not learn from the response
	// that another workspace's keyspace exists.
	if req.KeyspaceId != nil {
		keySpace, err := db.Query.FindKeySpaceByID(ctx, h.DB.RO(), *req.KeyspaceId)
		if err != nil {
			if db.IsNotFound(err) {
				return fault.New("keyspace not found",
					fault.Code(codes.Data.KeyAuth.NotFound.URN()),
					fault.Internal("keyspace not found"),
					fault.Public("The specified keyspace was not found."),
				)
			}
			return fault.Wrap(err, fault.Internal("failed to find keyspace"))
		}

		if keySpace.WorkspaceID != principal.WorkspaceID {
			return fault.New("keyspace not found",
				fault.Code(codes.Data.KeyAuth.NotFound.URN()),
				fault.Internal("keyspace belongs to different workspace, masking as 404"),
				fault.Public("The specified keyspace was not found."),
			)
		}
	}

	// An explicit image wins over git metadata and deploys as it is. Otherwise
	// this is a git build, where an empty branch and sha let the worker fall
	// back to the app's default branch.
	if req.DockerImage != "" {
		createReq.Source = &hydrav1.DeployCreateRequest_Image{
			Image: &hydrav1.CreateImageSource{Image: req.DockerImage},
		}
	} else {
		// nolint: exhaustruct // only the fields the caller provided are set
		commit := &ctrlv1.GitCommitInfo{
			Branch: req.Branch,
		}
		if req.GitCommit != nil {
			commit.CommitSha = ptr.SafeDeref(req.GitCommit.CommitSha)
			commit.CommitMessage = ptr.SafeDeref(req.GitCommit.CommitMessage)
			commit.AuthorHandle = ptr.SafeDeref(req.GitCommit.AuthorHandle)
			commit.AuthorAvatarUrl = ptr.SafeDeref(req.GitCommit.AuthorAvatarUrl)
			commit.Timestamp = ptr.SafeDeref(req.GitCommit.Timestamp)
		}
		createReq.Source = &hydrav1.DeployCreateRequest_Git{
			Git: &hydrav1.CreateGitSource{Commit: commit, PrNumber: 0},
		}
	}

	// The create is submitted one-way below, so this is the only place a caller
	// can be told its workspace may not deploy.
	if err := deployment.EnsureWorkspaceCanDeploy(ctx, h.DB, principal.WorkspaceID); err != nil {
		return err
	}

	// Send, not Request: the create resolves commits against GitHub and retries
	// transient failures for minutes, which no HTTP caller should wait through.
	// The worker re-checks the gates above, so a workspace that loses its
	// entitlement in between is still refused, asynchronously.
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
		Data: openapi.V2DeployCreateDeploymentResponseData{
			DeploymentId: deploymentID,
		},
	})
}
