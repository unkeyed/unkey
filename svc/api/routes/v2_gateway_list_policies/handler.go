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
	"github.com/unkeyed/unkey/svc/api/internal/policyconfig"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2GatewayListPoliciesRequestBody
	Response = openapi.V2GatewayListPoliciesResponseBody
)

type Handler struct {
	DB db.Database
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/gateway.listPolicies"
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

	env, err := db.Query.FindEnvironmentByIdentifiers(ctx, h.DB.RO(), db.FindEnvironmentByIdentifiersParams{
		WorkspaceID: principal.AuthorizedWorkspaceID,
		Project:     req.Project,
		App:         req.App,
		Environment: req.Environment,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New(
				"environment not found",
				fault.Code(codes.Data.Environment.NotFound.URN()),
				fault.Internal("environment not found"),
				fault.Public("The requested environment does not exist."),
			)
		}
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve environment."),
		)
	}

	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   "*",
			Action:       rbac.ReadPolicies,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   env.ID,
			Action:       rbac.ReadPolicies,
		}),
		rbac.U(
			urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(env.ProjectID).App(env.AppID).Environment(env.ID).Gateway().Policy("*"),
			permissions.Read,
		),
	))
	if err != nil {
		return err
	}

	settings, err := db.Query.FindAppRuntimeSettingsByAppAndEnv(ctx, h.DB.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
		AppID:         env.AppID,
		EnvironmentID: env.ID,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return s.JSON(http.StatusOK, Response{
				Meta: openapi.Meta{RequestId: s.RequestID()},
				Data: []openapi.PolicyResponse{},
			})
		}
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("unable to read gateway policy config"),
			fault.Public("We're unable to list the policies."),
		)
	}
	// Frontline tolerates a broken blob by skipping it to stay up, but we
	// must surface the failure rather than pass an unreadable
	// config off as "no policies".
	cfg, err := policyconfig.Parse(settings.SentinelConfig)
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("stored gateway policy config is not valid protojson"),
			fault.Public("We're unable to list the policies."),
		)
	}

	all, err := policyconfig.FromProto(cfg.GetPolicies())
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: all,
	})
}
