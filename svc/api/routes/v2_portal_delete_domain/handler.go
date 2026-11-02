package handler

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/internal/customdomain"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
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

	lookup, err := portaldomain.FindPortalAndDomain(ctx, h.DB, principal.AuthorizedWorkspaceID, req.Portal, req.DomainId)
	if err != nil {
		return err
	}

	portalURN := urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(lookup.Portal.ProjectID).Portal(lookup.Portal.ID)
	permissionQuery := rbac.U(portalURN.Domain(req.DomainId), permissions.Delete)
	if err = principal.Authorize(permissionQuery); err != nil {
		return apierrors.MaskInsufficientPermissionsWithVisibility(
			err,
			principal.Permissions,
			permissionQuery,
			rbac.Or(
				rbac.U(portalURN, permissions.Read),
				rbac.U(portalURN.Domain(req.DomainId), permissions.Read),
			),
			codes.Data.Portal.NotFound.URN(),
			portaldomain.PortalNotFoundMessage,
		)
	}
	if lookup.DomainErr != nil {
		return lookup.DomainErr
	}

	actor, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}

	_, err = h.CtrlClient.DeletePortalDomain(ctx, &ctrlv1.DeletePortalDomainRequest{
		WorkspaceId: principal.AuthorizedWorkspaceID,
		PortalId:    lookup.Portal.ID,
		DomainId:    lookup.Domain.ID,
		Actor:       actor,
	})
	// A concurrent delete got there first, which leaves what the caller asked for.
	if err != nil && connect.CodeOf(err) != connect.CodeNotFound {
		return customdomain.MapCtrlError(err, "delete portal domain")
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.EmptyResponse{},
	})
}
