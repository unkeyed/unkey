package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/db"
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

	found, err := portaldomain.ResolvePortal(ctx, h.DB, principal, req.Portal, portaldomain.Read)
	if err != nil {
		return err
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
