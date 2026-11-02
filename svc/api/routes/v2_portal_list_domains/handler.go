package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/array"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
	"github.com/unkeyed/unkey/svc/api/internal/portaldomain"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2PortalListDomainsRequestBody
	Response = openapi.V2PortalListDomainsResponseBody
)

type Handler struct {
	DB db.Database
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/portal.listDomains"
}

// Handle returns every domain on the portal unpaginated, since ctrl caps a
// workspace at a handful of portal domains.
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	found, err := portaldomain.FindPortal(ctx, h.DB, principal.AuthorizedWorkspaceID, req.Portal)
	if err != nil {
		return err
	}

	portalURN := urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(found.ProjectID).Portal(found.ID)
	permissionQuery := rbac.U(portalURN.Domain("*"), permissions.Read)
	if err = principal.Authorize(permissionQuery); err != nil {
		return apierrors.MaskInsufficientPermissionsWithVisibility(
			err,
			principal.Permissions,
			permissionQuery,
			rbac.U(portalURN, permissions.Read),
			codes.Data.Portal.NotFound.URN(),
			portaldomain.PortalNotFoundMessage,
		)
	}

	rows, err := db.Query.ListPortalDomainsByPortal(ctx, h.DB.RO(), db.ListPortalDomainsByPortalParams{
		PortalID:    found.ID,
		WorkspaceID: principal.AuthorizedWorkspaceID,
	})
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error listing portal domains"),
			fault.Public("Failed to list domains."),
		)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: array.Map(rows, func(row db.ListPortalDomainsByPortalRow) openapi.PortalDomain {
			return portaldomain.ToResponse(db.FindPortalDomainByIdRow(row))
		}),
	})
}
