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
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/internal/customdomain"
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

	found, err := portaldomain.ResolvePortal(ctx, h.DB, principal, req.Portal, portaldomain.Update)
	if err != nil {
		return err
	}

	row, err := portaldomain.FindDomain(ctx, h.DB, principal.AuthorizedWorkspaceID, found.ID, req.DomainId)
	if err != nil {
		return err
	}

	if row.VerificationStatus == db.PortalDomainsVerificationStatusVerified {
		return domaingate.AlreadyVerified(row.Domain)
	}

	actor, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}

	_, err = h.CtrlClient.RetryVerification(ctx, &ctrlv1.RetryPortalDomainVerificationRequest{
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
		// The verified guard above already passed, so this is the row verifying
		// between that read and ctrl's re-check.
		case connect.CodeFailedPrecondition:
			return fault.Wrap(err,
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Internal(fmt.Sprintf("ctrl rejected the verification retry for portal domain %s: %s", row.ID, err.Error())),
				fault.Public("The domain is already verified. No action is needed."),
			)
		default:
			return customdomain.MapCtrlError(err, "verify portal domain")
		}
	}

	return s.JSON(http.StatusAccepted, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.EmptyResponse{},
	})
}
