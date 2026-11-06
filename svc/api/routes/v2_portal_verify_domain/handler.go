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
	Request  = openapi.V2PortalVerifyDomainRequestBody
	Response = openapi.V2PortalVerifyDomainResponseBody
)

type Handler struct {
	DB         db.Database
	CtrlClient ctrl.PortalDomainServiceClient
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/portal.verifyDomain"
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
	found := lookup.Portal

	portalURN := urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(found.ProjectID).Portal(found.ID)
	permissionQuery := rbac.U(portalURN.Domain(req.DomainId), permissions.Write)
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
	row := lookup.Domain

	// Nothing to retry, and the caller's goal already holds.
	if row.VerificationStatus == db.PortalDomainsVerificationStatusVerified {
		return accepted(s)
	}

	actor, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}

	_, err = h.CtrlClient.RetryPortalDomainVerification(ctx, &ctrlv1.RetryPortalDomainVerificationRequest{
		WorkspaceId: principal.AuthorizedWorkspaceID,
		PortalId:    found.ID,
		DomainId:    row.ID,
		Actor:       actor,
	})
	if err != nil {
		//nolint:exhaustive // all other Connect error codes fall through to the generic mapping
		switch connect.CodeOf(err) {
		case connect.CodeNotFound:
			return portaldomain.DomainNotFound(fmt.Sprintf("ctrl has no domain %s on portal %s: %s", row.ID, found.ID, err.Error()))
		// The row verified between the read above and ctrl's re-check.
		case connect.CodeFailedPrecondition:
			return accepted(s)
		default:
			return customdomain.MapCtrlError(err, "verify portal domain")
		}
	}

	return accepted(s)
}

func accepted(s *zen.Session) error {
	return s.JSON(http.StatusAccepted, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.EmptyResponse{},
	})
}
