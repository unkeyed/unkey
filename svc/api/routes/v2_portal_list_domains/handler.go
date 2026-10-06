package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
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

	found, err := portaldomain.ResolvePortal(ctx, h.DB, principal, req.Portal, portaldomain.Read)
	if err != nil {
		return err
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

	data := make([]openapi.PortalDomain, 0, len(rows))
	for _, row := range rows {
		data = append(data, portaldomain.ToResponse(db.FindPortalDomainByIdRow(row)))
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: data,
	})
}
