package handler

import (
	"context"
	"net/http"

	restateingress "github.com/restatedev/sdk-go/ingress"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/imageref"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
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

	// The worker rejects a bad image too, with a coarser message.
	if err := imageref.Validate(req.DockerImage); err != nil {
		return err
	}

	actorInfo, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}

	// The CLI built the image itself, so the commit is metadata to record, not
	// something to build. The branch alone still scopes sibling dedup.
	// nolint: exhaustruct // optional fields, only what the caller sent
	commit := &ctrlv1.GitCommitInfo{Branch: req.Branch}
	if req.GitCommit != nil {
		commit.CommitSha = ptr.SafeDeref(req.GitCommit.CommitSha)
		commit.CommitMessage = ptr.SafeDeref(req.GitCommit.CommitMessage)
		commit.AuthorHandle = ptr.SafeDeref(req.GitCommit.AuthorHandle)
		commit.AuthorAvatarUrl = ptr.SafeDeref(req.GitCommit.AuthorAvatarUrl)
		commit.Timestamp = ptr.SafeDeref(req.GitCommit.Timestamp)
	}

	// nolint: exhaustruct // the source oneof is set above
	createReq := &hydrav1.DeployCreateRequest{
		ProjectId:     row.ProjectID,
		AppId:         row.AppID,
		EnvironmentId: environment.ID,
		Source: &hydrav1.DeployCreateRequest_Image{
			Image: &hydrav1.CreateImageSource{Image: req.DockerImage, Commit: commit},
		},
		Decision:      hydrav1.CreateDecision_CREATE_DECISION_DEPLOY,
		Trigger:       deployment.TriggerFromClient(s),
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

	deploymentID, err := deployment.Create(ctx, h.Restate, createReq)
	if err != nil {
		return err
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
