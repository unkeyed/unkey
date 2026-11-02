package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
	"github.com/unkeyed/unkey/svc/api/internal/portaldomain"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2PortalGetDomainRequestBody
	Response = openapi.V2PortalGetDomainResponseBody
)

type Handler struct {
	DB db.Database
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/portal.getDomain"
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

	lookup, err := portaldomain.FindPortalAndDomain(ctx, h.DB, principal.AuthorizedWorkspaceID, req.Portal, req.DomainId)
	if err != nil {
		return err
	}

	portalURN := urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(lookup.Portal.ProjectID).Portal(lookup.Portal.ID)
	permissionQuery := rbac.U(portalURN.Domain(req.DomainId), permissions.Read)
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
	if lookup.DomainErr != nil {
		return lookup.DomainErr
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: portaldomain.ToResponse(lookup.Domain),
	})
}
