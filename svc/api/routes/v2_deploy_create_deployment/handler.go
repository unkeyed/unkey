package handler

import (
	"context"
	"net/http"
	"strings"

	restateingress "github.com/restatedev/sdk-go/ingress"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/imageref"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
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
		WorkspaceID: principal.AuthorizedWorkspaceID,
		ProjectSlug: req.Project,
		AppSlug:     req.App,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New(
				"project or app not found",
				fault.Code(codes.Data.Project.NotFound.URN()),
				fault.Internal("project or app not found"),
				fault.Public("The requested project or app does not exist."),
			)
		}
		return fault.Wrap(err, fault.Internal("failed to find project and app"))
	}

	environment, err := db.Query.FindEnvironmentByIdentifiers(ctx, h.DB.RO(), db.FindEnvironmentByIdentifiersParams{
		WorkspaceID: principal.AuthorizedWorkspaceID,
		Project:     req.Project,
		App:         req.App,
		Environment: req.EnvironmentSlug,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New(
				"environment not found",
				fault.Code(codes.Data.Environment.NotFound.URN()),
				fault.Internal("environment did not resolve"),
				fault.Public("The requested environment does not exist."),
			)
		}
		return fault.Wrap(err, fault.Internal("failed to resolve environment"))
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

	// CLI announces itself via X-Unkey-Client: unkey-cli/<version>.
	// Anything else (or absent) is attributed to the API.
	trigger := ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_API
	if strings.HasPrefix(s.Request().Header.Get("X-Unkey-Client"), "unkey-cli/") {
		trigger = ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_CLI
	}

	// ctrl rejects these too, but ctrlclient.HandleError replaces its message with a
	// generic one, so the reason reaches the caller only if the check also runs here.
	if err := imageref.Validate(req.DockerImage); err != nil {
		return err
	}

	actorInfo, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}

	// The id is the Restate object key the create runs on, so minting it here
	// lets the response name the deployment without waiting on the worker.
	deploymentID := uid.New(uid.DeploymentPrefix)

	// nolint: exhaustruct // the source oneof is set below
	createReq := &hydrav1.DeployCreateRequest{
		ProjectId:   row.ProjectID,
		AppId:       row.AppID,
		Environment: environment.ID,
		Source: &hydrav1.DeployCreateRequest_Image{
			Image: &hydrav1.CreateImageSource{Image: req.DockerImage},
		},
		Decision:      hydrav1.CreateDecision_CREATE_DECISION_DEPLOY,
		Trigger:       trigger,
		TriggeredBy:   principal.Subject.ID,
		TriggerReason: "",
		Actor:         actorInfo,
	}

	// Add optional keyspace ID for authentication. Verify the keyspace belongs
	// to the caller's workspace before attaching it; otherwise a root key for
	// one workspace could bind another workspace's keyspace into its
	// deployment's key-auth allowlist (cross-tenant isolation violation).
	if req.KeyspaceId != nil {
		keySpace, err := db.Query.FindKeySpaceByID(ctx, h.DB.RO(), *req.KeyspaceId)
		if err != nil {
			if db.IsNotFound(err) {
				return fault.New(
					"keyspace not found",
					fault.Code(codes.Data.KeyAuth.NotFound.URN()),
					fault.Internal("keyspace not found"),
					fault.Public("The specified keyspace was not found."),
				)
			}
			return fault.Wrap(err, fault.Internal("failed to find keyspace"))
		}

		if keySpace.WorkspaceID != principal.AuthorizedWorkspaceID {
			return fault.New(
				"keyspace not found",
				fault.Code(codes.Data.KeyAuth.NotFound.URN()),
				fault.Internal("keyspace belongs to different workspace, masking as 404"),
				fault.Public("The specified keyspace was not found."),
			)
		}
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
		createReq.Source = &hydrav1.DeployCreateRequest_Git{
			Git: &hydrav1.CreateGitSource{Commit: gitCommit, PrNumber: 0},
		}
	}

	// Request, not Send: the create is the only writer of the deployment row, so
	// awaiting it is what lets this response name a deployment a caller can
	// immediately read back. It also carries the worker's own gates, which is
	// where a workspace that may not deploy is refused.
	//
	// A timeout here does not undo the create. Restate keeps running it, so a
	// caller that gives up may still get a deployment.
	res, err := hydrav1.NewDeployServiceIngressClient(h.Restate, deploymentID).
		Create().
		Request(ctx, createReq)
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to submit deployment create to Restate"),
			fault.Public("Failed to create deployment."),
		)
	}
	if res.GetOutcome() == hydrav1.CreateOutcome_CREATE_OUTCOME_REJECTED {
		return deployment.RejectionFault(res.GetRejectionReason())
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
