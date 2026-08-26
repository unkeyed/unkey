package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/unkeyed/unkey/pkg/array"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/domain"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
	"github.com/unkeyed/unkey/svc/api/internal/pagination"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2DomainsListDomainsRequestBody
	Response = openapi.V2DomainsListDomainsResponseBody
)

type Handler struct {
	DB db.Database
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/domains.listDomains"
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

	if req.Environment == nil {
		if err = principal.Authorize(rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   "*",
			Action:       rbac.ReadDomain,
		})); err != nil {
			return err
		}
	}

	scope, err := h.resolveScope(ctx, principal.AuthorizedWorkspaceID, req)
	if err != nil {
		return err
	}

	if req.Environment != nil {
		if err = principal.Authorize(rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Environment,
				ResourceID:   "*",
				Action:       rbac.ReadDomain,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Environment,
				ResourceID:   scope.environmentID,
				Action:       rbac.ReadDomain,
			}),
			rbac.U(
				urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(scope.projectID).App(scope.appID).Environment(scope.environmentID).Domain("*"),
				permissions.Read,
			),
		)); err != nil {
			return apierrors.MaskInsufficientPermissionsAsNotFound(
				err,
				codes.Data.Environment.NotFound.URN(),
				"The requested environment does not exist.",
			)
		}
	}

	p := pagination.Parse(req.Limit, req.Cursor, 100)
	search := mysql.SearchContains(strings.TrimSpace(ptr.SafeDeref(req.Search)))

	rows, err := db.Query.ListCustomDomains(ctx, h.DB.RO(), db.ListCustomDomainsParams{
		WorkspaceID:   principal.AuthorizedWorkspaceID,
		ProjectID:     scope.projectID,
		AppID:         scope.appID,
		EnvironmentID: scope.environmentID,
		IDCursor:      p.Cursor,
		Search:        search,
		Limit:         p.FetchLimit(),
	})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve domains."),
		)
	}

	rows, pg := pagination.Paginate(rows, p, func(r db.ListCustomDomainsRow) string { return r.ID })

	data := array.Map(rows, func(row db.ListCustomDomainsRow) openapi.Domain {
		d := openapi.Domain{
			Id:                row.ID,
			Domain:            row.Domain,
			ProjectId:         row.ProjectID,
			AppId:             row.AppID,
			EnvironmentId:     row.EnvironmentID,
			Status:            domain.Status(row.VerificationStatus),
			VerificationError: nil,
			DnsRecords: domain.DnsRecords(domain.DnsRecordsInput{
				Domain:            row.Domain,
				TargetCname:       row.TargetCname,
				VerificationToken: row.VerificationToken,
				RoutingVerified:   row.CnameVerified,
				OwnershipVerified: row.OwnershipVerified,
			}),
			DomainConnect: nil,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     nil,
		}
		if row.DomainConnectProvider.Valid && row.DomainConnectUrl.Valid {
			d.DomainConnect = &openapi.DomainConnect{
				Provider: row.DomainConnectProvider.String,
				Url:      row.DomainConnectUrl.String,
			}
		}
		if row.VerificationError.Valid && row.VerificationError.String != "" {
			d.VerificationError = ptr.P(row.VerificationError.String)
		}
		if row.UpdatedAt.Valid {
			d.UpdatedAt = ptr.P(row.UpdatedAt.Int64)
		}
		return d
	})

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data:       data,
		Pagination: pg,
	})
}

// resolvedScope uses empty IDs to disable the corresponding database filters.
type resolvedScope struct {
	projectID     string
	appID         string
	environmentID string
}

// resolveScope uses globally unique IDs directly and resolves slugs under their parent scope.
// It returns not found when supplied parents do not contain the requested child.
func (h *Handler) resolveScope(ctx context.Context, workspaceID string, req Request) (resolvedScope, error) {
	var resolved resolvedScope
	if req.Project != nil {
		project, err := db.Query.FindProjectByIdOrSlug(ctx, h.DB.RO(), db.FindProjectByIdOrSlugParams{
			WorkspaceID: workspaceID,
			Project:     *req.Project,
		})
		if err != nil {
			if db.IsNotFound(err) {
				return resolvedScope{}, resourceNotFound("project", codes.Data.Project.NotFound.URN())
			}
			return resolvedScope{}, listDatabaseError(err)
		}
		resolved.projectID = project.ID
	}

	if req.App != nil {
		var app db.App
		var err error
		if resolved.projectID != "" {
			app, err = db.Query.FindAppByProjectAndIdOrSlug(ctx, h.DB.RO(), db.FindAppByProjectAndIdOrSlugParams{
				WorkspaceID: workspaceID,
				Project:     resolved.projectID,
				App:         *req.App,
			})
		} else {
			app, err = db.Query.FindAppById(ctx, h.DB.RO(), *req.App)
			if err == nil && app.WorkspaceID != workspaceID {
				return resolvedScope{}, resourceNotFound("app", codes.Data.App.NotFound.URN())
			}
		}
		if err != nil {
			if db.IsNotFound(err) {
				return resolvedScope{}, resourceNotFound("app", codes.Data.App.NotFound.URN())
			}
			return resolvedScope{}, listDatabaseError(err)
		}
		resolved.projectID = app.ProjectID
		resolved.appID = app.ID
	}

	if req.Environment != nil {
		var environment db.Environment
		var err error
		if resolved.appID != "" {
			environment, err = db.Query.FindEnvironmentByIdentifiers(ctx, h.DB.RO(), db.FindEnvironmentByIdentifiersParams{
				WorkspaceID: workspaceID,
				Project:     resolved.projectID,
				App:         resolved.appID,
				Environment: *req.Environment,
			})
		} else {
			environment, err = db.Query.FindEnvironmentById(ctx, h.DB.RO(), *req.Environment)
			if err == nil && (environment.WorkspaceID != workspaceID || (resolved.projectID != "" && environment.ProjectID != resolved.projectID)) {
				return resolvedScope{}, resourceNotFound("environment", codes.Data.Environment.NotFound.URN())
			}
		}
		if err != nil {
			if db.IsNotFound(err) {
				return resolvedScope{}, resourceNotFound("environment", codes.Data.Environment.NotFound.URN())
			}
			return resolvedScope{}, listDatabaseError(err)
		}
		resolved.projectID = environment.ProjectID
		resolved.appID = environment.AppID
		resolved.environmentID = environment.ID
	}

	return resolved, nil
}

// resourceNotFound keeps scope lookup errors consistent across resource types.
func resourceNotFound(resource string, code codes.URN) error {
	return fault.New(
		resource+" not found",
		fault.Code(code),
		fault.Internal(resource+" not found"),
		fault.Public("The requested "+resource+" does not exist."),
	)
}

// listDatabaseError prevents database details from reaching the public response.
func listDatabaseError(err error) error {
	return fault.Wrap(
		err,
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Internal("database error"),
		fault.Public("Failed to retrieve domains."),
	)
}
