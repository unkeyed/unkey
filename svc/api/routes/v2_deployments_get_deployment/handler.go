package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/deployment"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2DeploymentsGetDeploymentRequestBody
	Response = openapi.V2DeploymentsGetDeploymentResponseBody
)

type Handler struct {
	DB db.Database
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/deployments.getDeployment"
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

	dep, err := db.Query.FindDeploymentById(ctx, h.DB.RO(), req.DeploymentId)
	if err != nil && !db.IsNotFound(err) {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve deployment."),
		)
	}

	// FindDeploymentById is not workspace-scoped, so a match in another workspace
	// is masked as not found to avoid leaking a deployment's existence.
	if db.IsNotFound(err) || dep.WorkspaceID != principal.AuthorizedWorkspaceID {
		return fault.New(
			"deployment not found",
			fault.Code(codes.Data.Deployment.NotFound.URN()),
			fault.Internal("deployment not found or belongs to another workspace"),
			fault.Public("The requested deployment does not exist."),
		)
	}

	err = principal.Authorize(rbac.U(
		urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(dep.ProjectID).App(dep.AppID).Environment(dep.EnvironmentID).Deployment(dep.ID),
		permissions.Read,
	))
	if err != nil {
		return fault.New(
			"deployment not found",
			fault.Code(codes.Data.Deployment.NotFound.URN()),
			fault.Internal("authorization failed; returning not found to avoid leaking deployment existence"),
			fault.Public("The requested deployment does not exist."),
		)
	}

	states, err := db.Query.ListDeploymentEnvAndAppState(ctx, h.DB.RO(), db.ListDeploymentEnvAndAppStateParams{
		WorkspaceID:   principal.AuthorizedWorkspaceID,
		DeploymentIds: []string{dep.ID},
	})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve deployment."),
		)
	}
	var state db.ListDeploymentEnvAndAppStateRow //nolint:exhaustruct // zero value when the join misses
	if len(states) > 0 {
		state = states[0]
	}

	steps, err := db.Query.ListDeploymentStepsByIds(ctx, h.DB.RO(), db.ListDeploymentStepsByIdsParams{
		WorkspaceID:   principal.AuthorizedWorkspaceID,
		DeploymentIds: []string{dep.ID},
	})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve deployment."),
		)
	}

	domains, err := db.Query.ListDeploymentDomains(ctx, h.DB.RO(), db.ListDeploymentDomainsParams{
		WorkspaceID:  principal.AuthorizedWorkspaceID,
		DeploymentID: dep.ID,
	})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve deployment."),
		)
	}

	regions, err := db.Query.ListDeploymentRegions(ctx, h.DB.RO(), db.ListDeploymentRegionsParams{
		WorkspaceID:  principal.AuthorizedWorkspaceID,
		DeploymentID: dep.ID,
	})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve deployment."),
		)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: deployment.ToResponse(deployment.Input{
			Deployment: db.ListDeploymentsRow{
				ID:                       dep.ID,
				Source:                   dep.Source,
				ImageRequested:           dep.ImageRequested,
				ImageResolved:            dep.ImageResolved,
				GitCommitSha:             dep.GitCommitSha,
				GitBranch:                dep.GitBranch,
				GitCommitMessage:         dep.GitCommitMessage,
				GitCommitAuthorHandle:    dep.GitCommitAuthorHandle,
				GitCommitAuthorAvatarUrl: dep.GitCommitAuthorAvatarUrl,
				GitCommitTimestamp:       dep.GitCommitTimestamp,
				CpuMillicores:            dep.CpuMillicores,
				MemoryMib:                dep.MemoryMib,
				StorageMib:               dep.StorageMib,
				DesiredState:             dep.DesiredState,
				Command:                  dep.Command,
				Port:                     dep.Port,
				ShutdownSignal:           dep.ShutdownSignal,
				UpstreamProtocol:         dep.UpstreamProtocol,
				Healthcheck:              dep.Healthcheck,
				PrNumber:                 dep.PrNumber,
				ForkRepositoryFullName:   dep.ForkRepositoryFullName,
				Status:                   dep.Status,
				Trigger:                  dep.Trigger,
				TriggeredBy:              dep.TriggeredBy,
				CreatedAt:                dep.CreatedAt,
				UpdatedAt:                dep.UpdatedAt,
			},
			State:   state,
			Steps:   steps,
			Regions: regions,
			Domains: domains,
		}),
	})
}
