package handler

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/domain/domaingate"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/internal/customdomain"
	"github.com/unkeyed/unkey/svc/api/internal/domain"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
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

	found, err := portaldomain.FindPortal(ctx, h.DB, principal.AuthorizedWorkspaceID, req.Portal)
	if err != nil {
		return err
	}

	portalURN := urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(found.ProjectID).Portal(found.ID)
	permissionQuery := rbac.U(portalURN.Domain("*"), permissions.Write)
	if err = principal.Authorize(permissionQuery); err != nil {
		return apierrors.MaskInsufficientPermissionsWithVisibility(
			err,
			principal.Permissions,
			permissionQuery,
			rbac.Or(
				rbac.U(portalURN, permissions.Read),
				rbac.U(portalURN.Domain("*"), permissions.Read),
			),
			codes.Data.Portal.NotFound.URN(),
			portaldomain.PortalNotFoundMessage,
		)
	}

	domainName, err := domaingate.ParseDomain(req.Domain)
	if err != nil {
		return err
	}

	attached, err := db.Query.CountPortalDomainsByWorkspace(ctx, h.DB.RO(), principal.AuthorizedWorkspaceID)
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to count the workspace's portal domains."),
		)
	}

	if err = domaingate.CheckPortalAllowance(attached); err != nil {
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
		// ctrl re-checks the portal and the cap on the primary, so a request that
		// raced past the reads above gets the same response as one caught there.
		//nolint:exhaustive // all other Connect error codes fall through to the generic mapping
		switch connect.CodeOf(err) {
		case connect.CodeNotFound:
			return portaldomain.PortalNotFound(fmt.Sprintf("portal %s deleted before the domain was added", found.ID))
		case connect.CodeResourceExhausted:
			return domaingate.CheckPortalAllowance(domaingate.PortalDomainsMax)
		}
		return customdomain.MapCtrlError(err, "create portal domain")
	}

	data := openapi.V2PortalCreateDomainResponseData{
		DomainId: res.GetDomainId(),
		DnsRecords: domain.DnsRecords(domain.DnsRecordsInput{
			Domain:            domainName,
			TargetCname:       res.GetTargetCname(),
			VerificationToken: res.GetVerificationToken(),
			RoutingVerified:   false,
			OwnershipVerified: false,
		}),
		DomainConnect: nil,
	}
	if dc := res.GetDomainConnect(); dc != nil {
		data.DomainConnect = &openapi.DomainConnect{
			Provider: dc.GetProvider(),
			Url:      dc.GetUrl(),
		}
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: data,
	})
}
