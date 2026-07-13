package handler

import (
	"context"
	"net/http"
	"slices"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/pagination"
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
	))
	if err != nil {
		return err
	}

	// A missing settings row means the environment was never configured;
	// that's an empty policy list, not an error. On NotFound the row is its
	// zero value, so the blob comes back nil either way.
	settings, err := db.Query.FindAppRuntimeSettingsByAppAndEnv(ctx, h.DB.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
		AppID:         env.AppID,
		EnvironmentID: env.ID,
	})
	if err != nil && !db.IsNotFound(err) {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("unable to read sentinel config"),
			fault.Public("We're unable to list the policies."),
		)
	}
	// An undecodable blob is an error: frontline skips broken configs to
	// keep serving traffic, but a management API must not report "no
	// policies" for an environment whose config it cannot read.
	cfg, err := policyconfig.Parse(settings.AppRuntimeSetting.SentinelConfig)
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("stored sentinel config is not valid protojson"),
			fault.Public("We're unable to list the policies."),
		)
	}

	all, err := mapPoliciesFromProto(cfg.GetPolicies())
	if err != nil {
		return err
	}

	// Policies live in one ordered blob, so pagination is a slice window:
	// the cursor resolves to its index and the window over-fetches one item,
	// matching the id >= cursor look-ahead convention of the db-backed lists.
	p := pagination.Parse(req.Limit, req.Cursor, 50)
	start := 0
	if p.Cursor != "" {
		start = slices.IndexFunc(all, func(policy openapi.PolicyResponse) bool { return policy.Id == p.Cursor })
		if start < 0 {
			return fault.New(
				"cursor not found",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("cursor id not present in stored policies"),
				fault.Public("The provided cursor is invalid or has expired."),
			)
		}
	}

	window := all[start:min(start+int(p.FetchLimit()), len(all))]
	page, pg := pagination.Paginate(window, p, func(policy openapi.PolicyResponse) string { return policy.Id })

	return s.JSON(http.StatusOK, Response{
		Meta:       openapi.Meta{RequestId: s.RequestID()},
		Data:       page,
		Pagination: pg,
	})
}
