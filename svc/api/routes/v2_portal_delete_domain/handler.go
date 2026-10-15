package handler

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/internal/customdomain"
	"github.com/unkeyed/unkey/svc/api/internal/portaldomain"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2PortalDeleteDomainRequestBody
	Response = openapi.V2PortalDeleteDomainResponseBody
)

type Handler struct {
	DB         db.Database
	CtrlClient ctrl.PortalDomainServiceClient
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/portal.deleteDomain"
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

	found, err := portaldomain.ResolvePortal(ctx, h.DB, principal, req.Portal, portaldomain.Update)
	if err != nil {
		return err
	}

	row, err := portaldomain.FindDomain(ctx, h.DB, principal.AuthorizedWorkspaceID, found.ID, req.DomainId)
	if err != nil {
		return err
	}

	actor, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}

	_, err = h.CtrlClient.DeletePortalDomain(ctx, &ctrlv1.DeletePortalDomainRequest{
		WorkspaceId: principal.AuthorizedWorkspaceID,
		PortalId:    found.ID,
		DomainId:    row.ID,
		Actor:       actor,
	})
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return portaldomain.DomainNotFound(fmt.Sprintf("ctrl has no domain %s on portal %s: %s", row.ID, found.ID, err.Error()))
		}
		return customdomain.MapCtrlError(err, "delete portal domain")
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.EmptyResponse{},
	})
}
