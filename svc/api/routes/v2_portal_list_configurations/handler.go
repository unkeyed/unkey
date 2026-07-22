package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/portalconfig"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2PortalListConfigurationsRequestBody
	Response = openapi.V2PortalListConfigurationsResponseBody
)

// Handler implements zen.Route for listing a workspace's portal configurations.
// It is an operator action authenticated by a root key; only configurations in
// the root key's workspace are returned.
type Handler struct {
	DB db.Database
}

func (h *Handler) Method() string { return "POST" }
func (h *Handler) Path() string   { return "/v2/portal.listConfigurations" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	rows, err := db.Query.ListPortalConfigsByWorkspace(ctx, h.DB.RO(), principal.WorkspaceID)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error listing portal configs"),
			fault.Public("Failed to list portal configurations."),
		)
	}

	configs := make([]openapi.PortalConfiguration, len(rows))
	for i, row := range rows {
		configs[i] = portalconfig.ToResponse(row.PortalConfiguration, row.LogoUrl, row.PrimaryColor)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: configs,
	})
}
