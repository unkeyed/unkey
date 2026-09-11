package logdrains

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
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Get struct {
	DB db.Database
}

func (h *Get) Method() string { return http.MethodPost }
func (h *Get) Path() string   { return "/v2/logdrains.getLogdrain" }
func (h *Get) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	req, err := zen.BindBody[openapi.LogdrainIdRequest](s)
	if err != nil {
		return err
	}
	if err := principal.Authorize(rbac.U(urn.V1{WorkspaceID: principal.AuthorizedWorkspaceID, Resource: "logdrains/" + req.LogdrainId}, permissions.Read)); err != nil {
		return err
	}
	row, err := db.Query.FindLogdrain(ctx, h.DB.RO(), db.FindLogdrainParams{WorkspaceID: principal.AuthorizedWorkspaceID, ID: req.LogdrainId})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.Wrap(err, fault.Code(codes.Data.Logdrain.NotFound.URN()), fault.Public("Log drain not found."))
		}
		return err
	}
	data, err := toPublic(row)
	if err != nil {
		return err
	}
	return s.JSON(http.StatusOK, openapi.LogdrainResponse{Meta: openapi.Meta{RequestId: s.RequestID()}, Data: data})
}
