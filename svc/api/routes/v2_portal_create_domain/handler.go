package handler

import (
	"context"
	"net/http"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/domain/domaingate"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/internal/customdomain"
	"github.com/unkeyed/unkey/svc/api/internal/domain"
	"github.com/unkeyed/unkey/svc/api/internal/portaldomain"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2PortalCreateDomainRequestBody
	Response = openapi.V2PortalCreateDomainResponseBody
)

type Handler struct {
	DB         db.Database
	CtrlClient ctrl.PortalDomainServiceClient
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/portal.createDomain"
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

	domainName, err := domaingate.ParseDomain(req.Domain)
	if err != nil {
		return err
	}

	actor, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}

	res, err := h.CtrlClient.AddPortalDomain(ctx, &ctrlv1.AddPortalDomainRequest{
		WorkspaceId: principal.AuthorizedWorkspaceID,
		PortalId:    found.ID,
		Domain:      domainName,
		Actor:       actor,
	})
	if err != nil {
		return customdomain.MapCtrlError(err, "create portal domain")
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2PortalCreateDomainResponseData{
			DomainId: res.GetDomainId(),
			DnsRecords: domain.DnsRecords(domain.DnsRecordsInput{
				Domain:            domainName,
				TargetCname:       res.GetTargetCname(),
				VerificationToken: res.GetVerificationToken(),
				RoutingVerified:   false,
				OwnershipVerified: false,
			}),
		},
	})
}
