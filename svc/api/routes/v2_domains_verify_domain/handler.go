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
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2DomainsVerifyDomainRequestBody
	Response = openapi.V2DomainsVerifyDomainResponseBody
)

type Handler struct {
	DB         db.Database
	CtrlClient ctrl.CustomDomainServiceClient
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/domains.verifyDomain"
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

	identifier := domaingate.CanonicalizeIdentifier(req.Domain)

	domain, err := db.Query.FindCustomDomainByIdentifier(ctx, h.DB.RO(), db.FindCustomDomainByIdentifierParams{
		WorkspaceID: principal.WorkspaceID,
		Domain:      identifier,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.Wrap(
				err,
				fault.Code(codes.Data.Domain.NotFound.URN()),
				fault.Internal(fmt.Sprintf("no custom domain %q in workspace %s", identifier, principal.WorkspaceID)),
				fault.Public("The requested domain does not exist."),
			)
		}
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to look up custom domain %q: %s", identifier, err.Error())),
			fault.Public("Failed to retrieve domain."),
		)
	}

	if err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   "*",
			Action:       rbac.VerifyDomain,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   domain.EnvironmentID,
			Action:       rbac.VerifyDomain,
		}),
		rbac.U(
			urn.New().Workspace(principal.WorkspaceID).Project(domain.ProjectID).App(domain.AppID).Environment(domain.EnvironmentID).Domain(domain.ID),
			permissions.Write{},
		),
	)); err != nil {
		return apierrors.MaskInsufficientPermissionsAsNotFound(
			err,
			codes.Data.Domain.NotFound.URN(),
			"The requested domain does not exist.",
		)
	}

	if domain.VerificationStatus == db.CustomDomainsVerificationStatusVerified {
		return domaingate.AlreadyVerified(domain.Domain)
	}

	actor, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}

	_, err = h.CtrlClient.RetryVerification(ctx, &ctrlv1.RetryVerificationRequest{
		WorkspaceId: principal.WorkspaceID,
		ProjectId:   domain.ProjectID,
		Domain:      domain.Domain,
		Actor:       actor,
	})
	if err != nil {
		//nolint:exhaustive // all other Connect error codes fall through to the generic mapping
		switch connect.CodeOf(err) {
		case connect.CodeNotFound:
			return fault.Wrap(
				err,
				fault.Code(codes.Data.Domain.NotFound.URN()),
				fault.Internal(fmt.Sprintf("ctrl has no domain %q for project %s while row %s exists: %s", domain.Domain, domain.ProjectID, domain.ID, err.Error())),
				fault.Public("The requested domain does not exist."),
			)
		// Ctrl re-checks the verified guard the handler already passed, so a
		// precondition failure here means the state changed between the two checks.
		case connect.CodeFailedPrecondition:
			return fault.Wrap(
				err,
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Internal(fmt.Sprintf("ctrl rejected the verification retry for domain %s (%s) in project %s: %s", domain.ID, domain.Domain, domain.ProjectID, err.Error())),
				fault.Public("The domain is already verified. No action is needed."),
			)
		default:
			return customdomain.MapCtrlError(err, "verify custom domain")
		}
	}

	return s.JSON(http.StatusAccepted, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.EmptyResponse{},
	})
}
