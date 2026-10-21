package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
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

	found, err := portaldomain.FindPortal(ctx, h.DB, principal.AuthorizedWorkspaceID, req.Portal)
	if err != nil {
		return err
	}

	// Update also grants read, so a key that creates a domain can poll it. The
	// wildcard tuples are spelled out because a stored `*` matches literally.
	portalURN := urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(found.ProjectID).Portal(found.ID)
	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{ResourceType: rbac.Portal, ResourceID: "*", Action: rbac.ReadPortal}),
		rbac.T(rbac.Tuple{ResourceType: rbac.Portal, ResourceID: found.ID, Action: rbac.ReadPortal}),
		rbac.U(portalURN, permissions.Read),
		rbac.T(rbac.Tuple{ResourceType: rbac.Portal, ResourceID: "*", Action: rbac.UpdatePortal}),
		rbac.T(rbac.Tuple{ResourceType: rbac.Portal, ResourceID: found.ID, Action: rbac.UpdatePortal}),
		rbac.U(portalURN, permissions.Write),
	))
	if err != nil {
		return portaldomain.PortalNotFound(fmt.Sprintf("get domain denied for portal %s: %s", found.ID, fault.InternalMessage(err)))
	}

	row, err := portaldomain.FindDomain(ctx, h.DB, principal.AuthorizedWorkspaceID, found.ID, req.DomainId)
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: portaldomain.ToResponse(row),
	})
}
